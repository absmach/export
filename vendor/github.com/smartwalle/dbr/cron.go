package dbr

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"strings"
	"sync"
	"time"
)

const (
	defaultKey    = "default"
	defaultPrefix = "cron:job"
)

var (
	ErrJobExists = errors.New("dbr: job already exists")
)

type CronHandler func(name string)

type CronOption interface {
	Apply(*Cron)
}

type cronOptionFunc func(*Cron)

func (f cronOptionFunc) Apply(tm *Cron) {
	f(tm)
}

func WithCronRedSync(redSync *RedSync) CronOption {
	return cronOptionFunc(func(tm *Cron) {
		tm.redSync = redSync
	})
}

func WithCronLocation(location *time.Location) CronOption {
	return cronOptionFunc(func(tm *Cron) {
		tm.location = location
	})
}

// Cron 是基于 Redis 实现的一个分布式定时任务管理工具，主要是为了避免同一服务的多个实例同时执行同一定时任务，最小支持分级别的定时任务。
// 本工具不适用于多个实例需要同时执行同一定时任务的情境。
// 需要开启 Redis 的 Keyspace Notifications， 在 Redis 的配置文件中添加 notify-keyspace-events "Kx" 即可。
type Cron struct {
	prefix        string
	eventPrefix   string
	mutexPrefix   string
	consumePrefix string

	rPool   Pool
	redSync *RedSync

	streamKey      string
	streamGroup    string
	streamConsumer string

	mu      sync.RWMutex
	jobPool map[string]*Job

	location *time.Location
	parser   Parser
}

type Job struct {
	name     string
	schedule Schedule
	handler  CronHandler
}

func NewCron(key string, pool Pool, opts ...CronOption) *Cron {
	var c = &Cron{}
	c.rPool = pool

	key = strings.TrimSpace(key)
	if key == "" {
		key = defaultKey
	}
	c.prefix = key
	c.location = time.Local

	for _, opt := range opts {
		opt.Apply(c)
	}

	if c.redSync == nil {
		c.redSync = NewRedSync(pool)
	}

	c.prefix = fmt.Sprintf("%s:%s", defaultPrefix, c.prefix)
	c.eventPrefix = fmt.Sprintf("%s:event:", c.prefix)
	c.mutexPrefix = fmt.Sprintf("%s:mutex:", c.prefix)
	c.consumePrefix = fmt.Sprintf("%s:consume:", c.prefix)

	c.streamKey = fmt.Sprintf("%s:stream", c.prefix)
	c.streamGroup = fmt.Sprintf("%s:group", c.prefix)
	c.streamConsumer = fmt.Sprintf("%s:consumer", c.prefix)
	c.jobPool = make(map[string]*Job)

	c.parser = NewParser(Minute | Hour | Dom | Month | Dow | Descriptor)

	go c.subscribe()
	go c.consumerJob(c.streamKey, c.streamGroup, c.streamConsumer)
	return c
}

func (this *Cron) buildEventKey(name string) string {
	return fmt.Sprintf("%s%s", this.eventPrefix, name)
}

func (this *Cron) buildMutexKey(name string) string {
	return fmt.Sprintf("%s%s", this.mutexPrefix, name)
}

func (this *Cron) buildConsumeKey(name string) string {
	return fmt.Sprintf("%s%s", this.consumePrefix, name)
}

func (this *Cron) subscribe() {
	var key = fmt.Sprintf("__keyspace@*__:%s", this.buildEventKey("*"))

	var rSess = this.rPool.GetSession()
	defer rSess.Close()

	var pConn = &redis.PubSubConn{Conn: rSess.Conn()}
	pConn.PSubscribe(key)

	for {
		switch data := pConn.Receive().(type) {
		case error:
		case redis.Message:
			var channels = strings.Split(data.Channel, this.eventPrefix)
			if len(channels) < 2 {
				continue
			}
			var jobName = channels[1]
			if jobName == "" {
				continue
			}

			var action = string(data.Data)
			if action == "expired" {
				this.postJob(jobName)
			}
		}
	}
}

func (this *Cron) postJob(jobName string) {
	var rSess = this.rPool.GetSession()
	defer rSess.Close()
	rSess.XADD(this.streamKey, 0, "*", "job_name", jobName)
}

func (this *Cron) consumerJob(key, group, consumer string) {
	var rSess = this.rPool.GetSession()
	defer rSess.Close()

	rSess.XGROUPCREATE(key, group, "$", "MKSTREAM")

	for {
		var streams, err = rSess.XREADGROUP(group, consumer, 1, 0, key, ">").Streams()
		if err != nil {
			continue
		}

		for _, stream := range streams {
			rSess.XACK(key, group, stream.Id)
			rSess.XDEL(key, stream.Id)

			var jobName = stream.Fields["job_name"]
			if jobName == "" {
				continue
			}

			go this.handleJob(jobName)
		}
	}
}

func (this *Cron) handleJob(name string) {
	this.mu.RLock()
	var job = this.jobPool[name]
	this.mu.RUnlock()

	if job == nil || job.handler == nil {
		return
	}

	var mutexKey = this.buildMutexKey(job.name)
	var redMu = this.redSync.NewMutex(mutexKey, WithRetryCount(4))
	if err := redMu.Lock(); err != nil {
		return
	}

	// 重新激活任务
	var next, _ = this.runJob(job)
	var expiresIn int64 = 59000
	if next-expiresIn < 1000 {
		expiresIn = next - 1000
	}

	// 同一个任务在一分钟（最大时间）内只能被处理一次
	var consumeKey = this.buildConsumeKey(job.name)
	var rSess = this.rPool.GetSession()
	if rResult := rSess.SET(consumeKey, time.Now(), "PX", expiresIn, "NX"); rResult.MustString() == "OK" {
		go job.handler(job.name)
	}
	rSess.Close()

	redMu.Unlock()
}

func (this *Cron) Add(name, spec string, handler CronHandler) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	schedule, err := this.parser.Parse(spec)
	if err != nil {
		return err
	}

	this.mu.RLock()
	if _, ok := this.jobPool[name]; ok {
		this.mu.RUnlock()
		return ErrJobExists
	}
	this.mu.RUnlock()

	var job = &Job{}
	job.name = name
	job.schedule = schedule
	job.handler = handler

	this.mu.Lock()
	this.jobPool[name] = job
	this.mu.Unlock()

	_, err = this.runJob(job)
	return err
}

func (this *Cron) runJob(job *Job) (next int64, err error) {
	var rSess = this.rPool.GetSession()
	var key = this.buildEventKey(job.name)

	var now = time.Now().In(this.location)
	var nextTime = job.schedule.Next(now).In(this.location)
	next = (nextTime.UnixNano() - now.UnixNano()) / 1e6

	var rResult = rSess.SET(key, nextTime, "PX", next, "NX")
	rSess.Close()

	if rResult.Error != nil {
		return -1, rResult.Error
	}
	return next, nil
}

func (this *Cron) Remove(name string) {
	if name == "" {
		return
	}

	this.mu.Lock()
	delete(this.jobPool, name)
	this.mu.Unlock()

	var rSess = this.rPool.GetSession()
	defer rSess.Close()

	rSess.BeginTx()
	rSess.Send("DEL", this.buildEventKey(name))
	rSess.Send("DEL", this.buildConsumeKey(name))
	rSess.Commit()
}
