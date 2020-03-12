package dbr

import (
	"encoding/base64"
	"errors"
	"github.com/gomodule/redigo/redis"
	"math"
	"math/rand"
	"sync"
	"time"
)

// https://github.com/hjr265/redsync.go
// https://github.com/go-redsync/redsync

const (
	kDefaultRetryCount = 32
	kDefaultRetryDelay = 512 * time.Millisecond
)

var ErrLockFailed = errors.New("dbr: failed to acquire lock")

// --------------------------------------------------------------------------------
type RedSync struct {
	pools []Pool
}

func NewRedSync(pools ...Pool) *RedSync {
	var rs = &RedSync{}
	rs.pools = pools
	return rs
}

func (this *RedSync) NewMutex(name string, opts ...Option) *Mutex {
	return NewMutex(this.pools, name, opts...)
}

// --------------------------------------------------------------------------------
type Option interface {
	Apply(*Mutex)
}

type optionFunc func(*Mutex)

func (f optionFunc) Apply(m *Mutex) {
	f(m)
}

func WithExpire(expire time.Duration) Option {
	return optionFunc(func(m *Mutex) {
		m.expire = expire
	})
}

func WithRetryDelay(delay time.Duration) Option {
	return optionFunc(func(m *Mutex) {
		m.retryDelay = delay
	})
}

func WithRetryCount(count int) Option {
	return optionFunc(func(m *Mutex) {
		if count <= 0 {
			count = kDefaultRetryCount
		}
		m.retryCount = count
	})
}

func WithQuorum(quorum int) Option {
	return optionFunc(func(m *Mutex) {
		if quorum <= 0 {
			quorum = len(m.pools)/2 + 1
		}
		m.quorum = int(math.Min(float64(quorum), float64(len(m.pools))))
	})
}

func WithFactor(factor float64) Option {
	return optionFunc(func(m *Mutex) {
		m.factor = factor
	})
}

// --------------------------------------------------------------------------------
type Mutex struct {
	mu         sync.Mutex
	value      string
	name       string
	expire     time.Duration
	retryDelay time.Duration
	retryCount int
	quorum     int
	factor     float64
	pools      []Pool
}

func NewMutex(pools []Pool, name string, opts ...Option) *Mutex {
	var mu = &Mutex{}
	mu.name = name
	mu.expire = 10 * time.Second
	mu.retryCount = kDefaultRetryCount
	mu.retryDelay = kDefaultRetryDelay
	mu.factor = 0.01
	mu.quorum = len(pools)/2 + 1
	mu.pools = pools

	for _, opt := range opts {
		opt.Apply(mu)
	}
	return mu
}

func (this *Mutex) Lock() error {
	this.mu.Lock()
	defer this.mu.Unlock()

	value, err := this.getValue()

	if err != nil {
		return err
	}

	for i := 0; i <= this.retryCount; i++ {
		if i != 0 {
			time.Sleep(this.retryDelay)
		}

		start := time.Now()

		n := 0
		for _, pool := range this.pools {
			if this.acquire(pool, value) {
				n++
			}
		}

		until := time.Now().Add(this.expire - time.Now().Sub(start) - time.Duration(float64(this.expire)*this.factor) + 2*time.Millisecond)
		if n >= this.quorum && time.Now().Before(until) {
			this.value = value
			return nil
		}
		for _, pool := range this.pools {
			this.release(pool, value)
		}
	}
	return ErrLockFailed
}

func (this *Mutex) Unlock() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	n := 0
	for _, pool := range this.pools {
		if this.release(pool, this.value) {
			n++
		}
	}
	return n >= this.quorum
}

func (this *Mutex) getValue() (string, error) {
	b := make([]byte, 32)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	_, err := r.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (this *Mutex) acquire(p Pool, value string) bool {
	var rSess = p.GetSession()
	defer rSess.Close()

	var rResult = rSess.SET(this.name, value, "PX", int(this.expire/time.Millisecond), "NX")
	return rResult.Error == nil && rResult.MustString() == "OK"
}

var unlockScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

func (this *Mutex) release(p Pool, value string) bool {
	var rSess = p.GetSession()
	defer rSess.Close()

	status, err := unlockScript.Do(rSess.Conn(), this.name, value)
	return err == nil && status != 0
}

func (this *Mutex) Extend() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	n := 0
	for _, pool := range this.pools {
		if this.touch(pool, this.value, int(this.expire/time.Millisecond)) {
			n++
		}
	}
	return n >= this.quorum
}

var touchScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("SET", KEYS[1], ARGV[1], "XX", "PX", ARGV[2])
	else
		return "ERR"
	end
`)

func (this *Mutex) touch(pool Pool, value string, expire int) bool {
	var rSess = pool.GetSession()
	defer rSess.Close()

	status, err := redis.String(touchScript.Do(rSess.Conn(), this.name, value, expire))
	return err == nil && status != "ERR"
}
