package web

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

var loginLimit = rateLimit{}
var accountLimit = rateLimit{}
var registrationLimit = rateLimit{}
var postLimit = rateLimit{}
var threadLimit = rateLimit{}

func reloadRatelimits() {
	loginLimit.maximum = config.Cfg.RateLimit.Login.MaxAttempts
	loginLimit.resetTime = config.Cfg.RateLimit.Login.Timeout
	accountLimit.maximum = config.Cfg.RateLimit.Account.MaxAttempts
	accountLimit.resetTime = config.Cfg.RateLimit.Account.Timeout
	registrationLimit.maximum = config.Cfg.RateLimit.
		Registration.MaxAttempts
	registrationLimit.resetTime = config.Cfg.RateLimit.Registration.Timeout
	postLimit.maximum = config.Cfg.RateLimit.Post.MaxAttempts
	postLimit.resetTime = config.Cfg.RateLimit.Post.Timeout
	threadLimit.maximum = config.Cfg.RateLimit.Thread.MaxAttempts
	threadLimit.resetTime = config.Cfg.RateLimit.Thread.Timeout
}
