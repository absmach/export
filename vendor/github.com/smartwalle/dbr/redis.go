package dbr

import (
	"crypto/tls"
	"errors"
	"github.com/FZambia/sentinel"
	"github.com/gomodule/redigo/redis"
	"net"
	"time"
)

var (
	ErrInvalidConn = errors.New("dbr: invalid connection")
)

// --------------------------------------------------------------------------------

func DialReadTimeout(d time.Duration) redis.DialOption {
	return redis.DialReadTimeout(d)
}

func DialWriteTimeout(d time.Duration) redis.DialOption {
	return redis.DialWriteTimeout(d)
}

func DialConnectTimeout(d time.Duration) redis.DialOption {
	return redis.DialConnectTimeout(d)
}

func DialKeepAlive(d time.Duration) redis.DialOption {
	return redis.DialKeepAlive(d)
}

func DialNetDial(dial func(network, addr string) (net.Conn, error)) redis.DialOption {
	return redis.DialNetDial(dial)
}

func DialDatabase(db int) redis.DialOption {
	return redis.DialDatabase(db)
}

func DialPassword(password string) redis.DialOption {
	return redis.DialPassword(password)
}

func DialTLSConfig(c *tls.Config) redis.DialOption {
	return redis.DialTLSConfig(c)
}

func DialTLSSkipVerify(skip bool) redis.DialOption {
	return redis.DialTLSSkipVerify(skip)
}

func DialUseTLS(useTLS bool) redis.DialOption {
	return redis.DialUseTLS(useTLS)
}

// --------------------------------------------------------------------------------
func NewRedis(addr string, maxActive, maxIdle int, opts ...redis.DialOption) Pool {
	var dialFunc = func() (c redis.Conn, err error) {
		c, err = redis.Dial("tcp", addr, opts...)
		if err != nil {
			return nil, err
		}
		return c, err
	}

	var pool = &redis.Pool{}
	pool.MaxIdle = maxIdle
	pool.MaxActive = maxActive
	pool.Wait = true
	pool.IdleTimeout = 180 * time.Second
	pool.Dial = dialFunc

	return &redisPool{pool}
}

func NewRedisWithSentinel(addrs []string, masterName string, maxActive, maxIdle int, opts ...redis.DialOption) Pool {
	var s = &sentinel.Sentinel{
		Addrs:      addrs,
		MasterName: masterName,
		Dial: func(addr string) (redis.Conn, error) {
			timeout := 500 * time.Millisecond
			c, err := redis.Dial("tcp", addr, redis.DialReadTimeout(timeout), redis.DialWriteTimeout(timeout), redis.DialConnectTimeout(timeout))
			if err != nil {
				return nil, err
			}
			return c, nil
		},
	}

	var dialFunc = func() (c redis.Conn, err error) {
		addr, err := s.MasterAddr()
		c, err = redis.Dial("tcp", addr, opts...)

		if err != nil {
			return nil, err
		}
		return c, err
	}

	var testOnBorrow = func(c redis.Conn, t time.Time) error {
		if !sentinel.TestRole(c, "master") {
			return errors.New("role check failed")
		} else {
			return nil
		}
	}

	var pool = &redis.Pool{}
	pool.MaxIdle = maxIdle
	pool.MaxActive = maxActive
	pool.Wait = true
	pool.IdleTimeout = 180 * time.Second
	pool.Dial = dialFunc
	pool.TestOnBorrow = testOnBorrow

	return &redisPool{pool}
}

////////////////////////////////////////////////////////////////////////////////
type Pool interface {
	GetSession() *Session

	Release(s *Session)

	Close() error
}

type redisPool struct {
	*redis.Pool
}

func (this *redisPool) GetSession() *Session {
	var c = this.Pool.Get()
	return NewSession(c)
}

func (this *redisPool) Release(s *Session) {
	s.c.Close()
}

func (this *redisPool) Close() error {
	return this.Pool.Close()
}

////////////////////////////////////////////////////////////////////////////////
func NewSession(c redis.Conn) *Session {
	if c == nil {
		return nil
	}
	return &Session{c: c}
}

type Session struct {
	c redis.Conn
}

func (this *Session) Conn() redis.Conn {
	return this.c
}

func (this *Session) Close() error {
	if this.c != nil {
		return this.c.Close()
	}
	return nil
}

func (this *Session) Do(commandName string, args ...interface{}) *Result {
	if this.c != nil {
		return result(this.c.Do(commandName, args...))
	}
	return result(nil, ErrInvalidConn)
}

func (this *Session) Send(commandName string, args ...interface{}) *Result {
	var err = ErrInvalidConn
	if this.c != nil {
		err = this.c.Send(commandName, args...)
	}
	return result(nil, err)
}

////////////////////////////////////////////////////////////////////////////////
const kRedisSession = "redis_session"

type Context interface {
	Set(key string, value interface{})

	MustGet(key string) interface{}
}

func FromContext(ctx Context) *Session {
	return ctx.MustGet(kRedisSession).(*Session)
}

func ToContext(ctx Context, s *Session) {
	ctx.Set(kRedisSession, s)
}
