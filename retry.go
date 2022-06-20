package retry

import (
	"math"
	"time"
)

func New(p Retryer) Retry {
	return Retry{
		delayer: p,
	}
}

type Retryer interface {
	Delay(int) time.Duration
}

type AttemptsLimiter interface {
	MaxAttempts() int
}

type DeadlineLimiter interface {
	Deadline() time.Duration
}

type Retry struct {
	delayer Retryer
}

func (r Retry) Do(callback func(sequence int) error) Result {
	startingTime := time.Now()

	retry := true

	maxAttempts := math.MaxInt
	if limiter, ok := r.delayer.(AttemptsLimiter); ok {
		maxAttempts = limiter.MaxAttempts()
	}

	var deadline time.Duration = 1<<61 - 1
	if limiter, ok := r.delayer.(DeadlineLimiter); ok {
		deadline = limiter.Deadline()
	}

	var attempts int
	for retry && attempts < maxAttempts && time.Since(startingTime) < deadline {
		delay := r.delayer.Delay(attempts)

		if delay > 0 {
			time.Sleep(time.Duration(delay))
		}

		if err := callback(attempts); err == nil {
			retry = false
		}

		attempts += 1
	}

	duration := time.Since(startingTime)

	return Result{
		Attempts: attempts,
		Duration: duration,
		Success:  !retry,
	}
}

func WithInitialDelay(p Retryer, d time.Duration) Retryer {
	return initialDelay{
		Retryer: p,
		delay:   d,
	}
}

type initialDelay struct {
	Retryer
	delay time.Duration
}

func (w initialDelay) Delay(attempt int) time.Duration {
	if attempt == 0 {
		return w.delay
	}

	return w.Retryer.Delay(attempt)
}

func WithMaxAttempts(p Retryer, attempts int) Retryer {
	return maxAttempts{
		Retryer:  p,
		attempts: attempts,
	}
}

type maxAttempts struct {
	Retryer
	attempts int
}

func (w maxAttempts) MaxAttempts() int {
	return w.attempts
}
