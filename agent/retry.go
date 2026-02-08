package agent

import (
	"math/rand"
	"strings"
	"time"
)

const (
	defaultBaseBackoff = 200 * time.Millisecond
	defaultMaxBackoff  = 2 * time.Second

	// Rate limit specific backoff settings
	rateLimitBaseBackoff = 30 * time.Second
	rateLimitMaxBackoff  = 120 * time.Second
)

type RetryPolicy struct {
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration

	// RateLimitMaxAttempts is the number of retries specifically for rate limit errors.
	// If 0, defaults to 3.
	RateLimitMaxAttempts int
	// RateLimitBaseBackoff is the initial backoff for rate limit errors.
	// If 0, defaults to 30 seconds.
	RateLimitBaseBackoff time.Duration
	// RateLimitMaxBackoff is the maximum backoff for rate limit errors.
	// If 0, defaults to 120 seconds.
	RateLimitMaxBackoff time.Duration
}

func defaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:          1,
		BaseBackoff:          defaultBaseBackoff,
		MaxBackoff:           defaultMaxBackoff,
		RateLimitMaxAttempts: 3,
		RateLimitBaseBackoff: rateLimitBaseBackoff,
		RateLimitMaxBackoff:  rateLimitMaxBackoff,
	}
}

func normalizeRetryPolicy(in RetryPolicy) RetryPolicy {
	out := in
	if out.MaxAttempts < 1 {
		out.MaxAttempts = 1
	}
	if out.BaseBackoff <= 0 {
		out.BaseBackoff = defaultBaseBackoff
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = defaultMaxBackoff
	}
	if out.MaxBackoff < out.BaseBackoff {
		out.MaxBackoff = out.BaseBackoff
	}
	if out.RateLimitMaxAttempts <= 0 {
		out.RateLimitMaxAttempts = 3
	}
	if out.RateLimitBaseBackoff <= 0 {
		out.RateLimitBaseBackoff = rateLimitBaseBackoff
	}
	if out.RateLimitMaxBackoff <= 0 {
		out.RateLimitMaxBackoff = rateLimitMaxBackoff
	}
	if out.RateLimitMaxBackoff < out.RateLimitBaseBackoff {
		out.RateLimitMaxBackoff = out.RateLimitBaseBackoff
	}
	return out
}

// IsRateLimitError checks if an error is a rate limit error based on common patterns.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "rate_limit") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "too many requests")
}

// rateLimitBackoffForAttempt calculates backoff for rate limit errors with jitter.
func (p RetryPolicy) rateLimitBackoffForAttempt(retryNumber int) time.Duration {
	if retryNumber < 1 {
		retryNumber = 1
	}
	delay := p.RateLimitBaseBackoff
	for i := 1; i < retryNumber; i++ {
		delay = delay * 3 / 2 // 1.5x exponential for rate limits
		if delay >= p.RateLimitMaxBackoff {
			delay = p.RateLimitMaxBackoff
			break
		}
	}
	if delay > p.RateLimitMaxBackoff {
		delay = p.RateLimitMaxBackoff
	}

	// Add jitter (±20%) to prevent thundering herd
	jitter := time.Duration(rand.Float64()*0.4-0.2) * delay
	return delay + jitter
}

func (p RetryPolicy) backoffForAttempt(retryNumber int) time.Duration {
	if retryNumber < 1 {
		retryNumber = 1
	}
	delay := p.BaseBackoff
	for i := 1; i < retryNumber; i++ {
		delay *= 2
		if delay >= p.MaxBackoff {
			return p.MaxBackoff
		}
	}
	if delay > p.MaxBackoff {
		return p.MaxBackoff
	}
	return delay
}
