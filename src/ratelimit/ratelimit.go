package ratelimit

import (
	"errors"
	"sync"
	"time"

	"IB1/config"
	"IB1/util"
)

type rateLimit struct {
	tries     util.SafeMap[string, int]
	maximum   int
	resetTime int
	lastReset time.Time
	mutex     sync.Mutex
}

func (p *rateLimit) Try(key string) error {
	if p.maximum == 0 {
		return nil
	}
	p.mutex.Lock()
	if int(time.Since(p.lastReset).Seconds()) > p.resetTime {
		p.tries.Clear()
		p.lastReset = time.Now()
	}
	p.mutex.Unlock()
	v, ok := p.tries.Get(key)
	if !ok {
		v = 0
	}
	if v >= p.maximum {
		return errors.New("rate-limited")
	}
	p.tries.Set(key, v+1)
	return nil
}

var Login = rateLimit{}
var Account = rateLimit{}
var Registration = rateLimit{}
var Post = rateLimit{}
var Thread = rateLimit{}

func Reload() {
	Login.maximum = config.Cfg.RateLimit.Login.MaxAttempts
	Login.resetTime = config.Cfg.RateLimit.Login.Timeout
	Account.maximum = config.Cfg.RateLimit.Account.MaxAttempts
	Account.resetTime = config.Cfg.RateLimit.Account.Timeout
	Registration.maximum = config.Cfg.RateLimit.
		Registration.MaxAttempts
	Registration.resetTime = config.Cfg.RateLimit.Registration.Timeout
	Post.maximum = config.Cfg.RateLimit.Post.MaxAttempts
	Post.resetTime = config.Cfg.RateLimit.Post.Timeout
	Thread.maximum = config.Cfg.RateLimit.Thread.MaxAttempts
	Thread.resetTime = config.Cfg.RateLimit.Thread.Timeout
}
