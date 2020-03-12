package dbr

import (
	"github.com/gomodule/redigo/redis"
	"time"
)

const healthCheckPeriod = time.Minute

func (this *Session) Sub(onMessage func(channel, data string) error, channels ...string) error {
	if len(channels) == 0 || onMessage == nil {
		return nil
	}

	var psc = redis.PubSubConn{Conn: this.Conn()}
	if err := psc.Subscribe(redis.Args{}.AddFlat(channels)...); err != nil {
		return err
	}
	defer psc.Unsubscribe(redis.Args{}.AddFlat(channels)...)

	var done = make(chan error, 1)

	go func() {
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				if err := onMessage(v.Channel, string(v.Data)); err != nil {
					done <- err
				}
			case redis.Subscription:
				if v.Count <= 0 {
					done <- nil
				}
			case error:
				done <- v
			}
		}
	}()
	//
	//	var ticker = time.NewTicker(healthCheckPeriod)
	//	defer ticker.Stop()
	//
	//loop:
	//	for {
	//		select {
	//		case <-ticker.C:
	//			if err := psc.Ping(""); err != nil {
	//				break loop
	//			}
	//		case err := <-done:
	//			return err
	//		}
	//	}

	return <-done
}

func (this *Session) Pub(channel, data string) *Result {
	return this.Do("PUBLISH", channel, data)
}
