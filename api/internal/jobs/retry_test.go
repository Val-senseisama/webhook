package jobs_test

import (
	"testing"
	"time"

	"webhook/internal/jobs"

	"github.com/riverqueue/river/rivertype"
)

func TestRetryPolicy_BackoffOrder(t *testing.T) {
	policy := jobs.RetryPolicy{}

	// Expected minimum delay for each attempt (without jitter)
	want := []time.Duration{
		30 * time.Second,
		5 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
	}

	for attempt, minDelay := range want {
		t.Run(time.Duration(minDelay).String(), func(t *testing.T) {
			job := &rivertype.JobRow{Attempt: attempt + 1}
			next := policy.NextRetry(job)
			delay := time.Until(next)

			// Must be at least minDelay (jitter adds, never subtracts)
			if delay < minDelay {
				t.Errorf("attempt %d: got delay %v, want >= %v", attempt+1, delay, minDelay)
			}
			// Jitter is at most 10% of base delay, so cap is 1.1× the base
			maxDelay := minDelay + minDelay/10
			if delay > maxDelay+time.Second { // +1s for test execution time
				t.Errorf("attempt %d: got delay %v, want <= %v", attempt+1, delay, maxDelay)
			}
		})
	}
}

func TestRetryPolicy_BeyondMaxAttemptsClampsToLast(t *testing.T) {
	policy := jobs.RetryPolicy{}
	beyond := &rivertype.JobRow{Attempt: 99}
	delay := time.Until(policy.NextRetry(beyond))

	// Must be in the 2h bucket: [2h, 2h + 10% jitter]
	if delay < 2*time.Hour {
		t.Errorf("attempt 99 should clamp to 2h bucket, got %v", delay)
	}
	if delay > 2*time.Hour+13*time.Minute {
		t.Errorf("attempt 99 exceeds 2h + max jitter, got %v", delay)
	}
}

func TestRetryPolicy_DelaysAreMonotonicallyIncreasing(t *testing.T) {
	policy := jobs.RetryPolicy{}
	var prev time.Duration
	for attempt := 1; attempt <= 4; attempt++ {
		job := &rivertype.JobRow{Attempt: attempt}
		delay := time.Until(policy.NextRetry(job))
		if attempt > 1 && delay <= prev {
			t.Errorf("delay for attempt %d (%v) must be > attempt %d (%v)",
				attempt, delay, attempt-1, prev)
		}
		prev = delay
	}
}

func TestRetryPolicy_NextRetryIsInFuture(t *testing.T) {
	policy := jobs.RetryPolicy{}
	before := time.Now()
	for attempt := 1; attempt <= 6; attempt++ {
		job := &rivertype.JobRow{Attempt: attempt}
		next := policy.NextRetry(job)
		if !next.After(before) {
			t.Errorf("attempt %d: NextRetry returned past time %v", attempt, next)
		}
	}
}
