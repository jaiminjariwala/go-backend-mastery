package ratelimit

import "testing"

func TestBucket(t *testing.T) {
	r, err := Run(Config{Requests: 10, IntervalMS: 100, Capacity: 2, Rate: 2})
	if err != nil {
		t.Fatal(err)
	}
	if r.Events[2].Status != 429 || r.Events[2].RetryMS != 300 {
		t.Fatal("expected empty bucket and 300ms retry")
	}
	if r.Events[5].Status != 200 {
		t.Fatal("token should refill exactly at 500ms")
	}
	for _, e := range r.Events {
		if e.After < 0 || e.After > 2 || e.Accepted+e.Rejected != e.Request {
			t.Fatal("invalid bucket state")
		}
	}
}
func TestSlowTrafficAndCap(t *testing.T) {
	r, _ := Run(Config{Requests: 5, IntervalMS: 2000, Capacity: 2, Rate: 20})
	for _, e := range r.Events {
		if !e.Allowed || e.Before > 2 {
			t.Fatal("refill must cap at capacity")
		}
	}
}
func TestInvalid(t *testing.T) {
	if _, err := Run(Config{}); err == nil {
		t.Fatal("expected validation error")
	}
}
