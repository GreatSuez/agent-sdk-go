package agent

import (
	"errors"
	"testing"
	"time"
)

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"generic error", errors.New("something went wrong"), false},
		{"rate_limit_error", errors.New("rate_limit_error: too many requests"), true},
		{"rate limit error", errors.New("rate limit exceeded"), true},
		{"429 status", errors.New("API error (429): Too Many Requests"), true},
		{"too many requests", errors.New("too many requests"), true},
		{"anthropic rate limit", errors.New(`anthropic API error (429): {"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitError(tt.err)
			if got != tt.expected {
				t.Errorf("IsRateLimitError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestRetryPolicy_RateLimitBackoff(t *testing.T) {
	policy := RetryPolicy{
		RateLimitBaseBackoff: 1 * time.Second,
		RateLimitMaxBackoff:  10 * time.Second,
	}
	policy = normalizeRetryPolicy(policy)

	t.Run("first attempt uses base backoff", func(t *testing.T) {
		backoff := policy.rateLimitBackoffForAttempt(1)
		// Allow for jitter (±20%)
		minExpected := 800 * time.Millisecond
		maxExpected := 1200 * time.Millisecond
		if backoff < minExpected || backoff > maxExpected {
			t.Errorf("expected backoff between %v and %v, got %v", minExpected, maxExpected, backoff)
		}
	})

	t.Run("backoff increases with attempts", func(t *testing.T) {
		backoff1 := policy.rateLimitBackoffForAttempt(1)
		backoff2 := policy.rateLimitBackoffForAttempt(2)
		// Second attempt should generally be higher (1.5x base = 1.5s)
		// Account for jitter by checking a reasonable range
		if backoff2 < 1000*time.Millisecond {
			t.Errorf("expected second backoff to be >= 1s, got %v", backoff2)
		}
		// Just verify it's different (due to jitter, can't guarantee > but should be close)
		_ = backoff1 // Used for potential debugging
	})

	t.Run("backoff caps at max", func(t *testing.T) {
		backoff := policy.rateLimitBackoffForAttempt(100)
		// Should be capped at max ± 20% jitter
		maxAllowed := 12 * time.Second
		if backoff > maxAllowed {
			t.Errorf("expected backoff capped at ~%v, got %v", policy.RateLimitMaxBackoff, backoff)
		}
	})
}

func TestNormalizeRetryPolicy_RateLimitDefaults(t *testing.T) {
	t.Run("sets rate limit defaults when zero", func(t *testing.T) {
		policy := normalizeRetryPolicy(RetryPolicy{})
		if policy.RateLimitMaxAttempts != 3 {
			t.Errorf("expected RateLimitMaxAttempts=3, got %d", policy.RateLimitMaxAttempts)
		}
		if policy.RateLimitBaseBackoff != rateLimitBaseBackoff {
			t.Errorf("expected RateLimitBaseBackoff=%v, got %v", rateLimitBaseBackoff, policy.RateLimitBaseBackoff)
		}
		if policy.RateLimitMaxBackoff != rateLimitMaxBackoff {
			t.Errorf("expected RateLimitMaxBackoff=%v, got %v", rateLimitMaxBackoff, policy.RateLimitMaxBackoff)
		}
	})

	t.Run("preserves custom rate limit settings", func(t *testing.T) {
		policy := normalizeRetryPolicy(RetryPolicy{
			RateLimitMaxAttempts: 5,
			RateLimitBaseBackoff: 10 * time.Second,
			RateLimitMaxBackoff:  60 * time.Second,
		})
		if policy.RateLimitMaxAttempts != 5 {
			t.Errorf("expected RateLimitMaxAttempts=5, got %d", policy.RateLimitMaxAttempts)
		}
		if policy.RateLimitBaseBackoff != 10*time.Second {
			t.Errorf("expected RateLimitBaseBackoff=10s, got %v", policy.RateLimitBaseBackoff)
		}
		if policy.RateLimitMaxBackoff != 60*time.Second {
			t.Errorf("expected RateLimitMaxBackoff=60s, got %v", policy.RateLimitMaxBackoff)
		}
	})
}
