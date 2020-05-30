package mal

import (
	"time"
)

const (
	jikkanRateLimit = 2 * time.Second
)

func (c *Controller) rateLimiter() {
	if c.lastRequest.IsZero() {
		c.log.Debug("[MAL] [RateLimiter] first request")
		c.lastRequest = time.Now()
		return
	}
	dur := time.Since(c.lastRequest)
	if dur > jikkanRateLimit {
		c.log.Debugf("[MAL] [RateLimiter] last request was %v ago: do not wait", dur)
		c.lastRequest = time.Now()
		return
	}
	wait := jikkanRateLimit - dur
	c.log.Debugf("[MAL] [RateLimiter] last request was %v ago: waiting %v", dur, wait)
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-c.ctx.Done():
		c.log.Debugf("[MAL] [RateLimiter] context is not valid anymore: %v", c.ctx.Err())
	case <-t.C:
		c.log.Debug("[MAL] [RateLimiter] wait is over")
		c.lastRequest = time.Now()
	}
	return
}
