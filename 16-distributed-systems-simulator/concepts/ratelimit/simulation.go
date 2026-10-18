package ratelimit

import "fmt"

type Config struct {
	Requests   int `json:"requests"`
	IntervalMS int `json:"interval_ms"`
	Capacity   int `json:"capacity"`
	Rate       int `json:"rate"`
}
type Event struct {
	TimeMS   int     `json:"time_ms"`
	Request  int     `json:"request"`
	Before   float64 `json:"before"`
	After    float64 `json:"after"`
	Refilled float64 `json:"refilled"`
	Allowed  bool    `json:"allowed"`
	Status   int     `json:"status"`
	RetryMS  int     `json:"retry_ms"`
	Accepted int     `json:"accepted"`
	Rejected int     `json:"rejected"`
}
type Result struct {
	Events []Event `json:"events"`
}

// Run models one shared bucket. Tokens refill continuously with simulated time.
// Integer thousandths avoid floating-point boundary errors. Rejection consumes nothing.
func Run(c Config) (Result, error) {
	if c.Requests < 1 || c.Requests > 200 || c.IntervalMS < 10 || c.IntervalMS > 2000 || c.Capacity < 1 || c.Capacity > 20 || c.Rate < 1 || c.Rate > 20 {
		return Result{}, fmt.Errorf("use 1-200 users, 10-2000 ms spacing, 1-20 capacity, and 1-20 tokens per second")
	}
	result := Result{Events: []Event{}}
	tokens, last, accepted, rejected := c.Capacity*1000, 0, 0, 0
	for i := 1; i <= c.Requests; i++ {
		at := (i - 1) * c.IntervalMS
		old := tokens
		tokens += (at - last) * c.Rate
		if tokens > c.Capacity*1000 {
			tokens = c.Capacity * 1000
		}
		e := Event{TimeMS: at, Request: i, Before: float64(tokens) / 1000, Refilled: float64(tokens-old) / 1000}
		if tokens >= 1000 {
			tokens -= 1000
			accepted++
			e.Allowed = true
			e.Status = 200
		} else {
			rejected++
			e.Status = 429
			e.RetryMS = (1000 - tokens + c.Rate - 1) / c.Rate
		}
		e.After = float64(tokens) / 1000
		e.Accepted = accepted
		e.Rejected = rejected
		result.Events = append(result.Events, e)
		last = at
	}
	return result, nil
}
