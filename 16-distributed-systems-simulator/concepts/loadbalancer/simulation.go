package loadbalancer

import (
	"fmt"
	"sort"
)

type Server struct {
	Name      string `json:"name"`
	Healthy   bool   `json:"healthy"`
	ServiceMS int    `json:"service_ms"`
}
type Config struct {
	Algorithm  string   `json:"algorithm"`
	Users      int      `json:"users"`
	IntervalMS int      `json:"interval_ms"`
	Servers    []Server `json:"servers"`
}
type State struct {
	Server
	Assigned  int `json:"assigned"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
}
type Event struct {
	TimeMS   int     `json:"time_ms"`
	Request  int     `json:"request"`
	Kind     string  `json:"kind"`
	Server   string  `json:"server"`
	Reason   string  `json:"reason"`
	States   []State `json:"states"`
	Rejected int     `json:"rejected"`
}
type Result struct {
	Events []Event `json:"events"`
}
type pending struct{ end, request, server int }

// Run uses a simulated clock; it sends no real network traffic to server nodes.
func Run(c Config) (Result, error) {
	if c.Algorithm != "round_robin" && c.Algorithm != "least_connections" {
		return Result{}, fmt.Errorf("unknown algorithm")
	}
	if c.Users < 1 || c.Users > 200 || c.IntervalMS < 10 || c.IntervalMS > 1000 || len(c.Servers) < 1 || len(c.Servers) > 30 {
		return Result{}, fmt.Errorf("use 1-200 users, 10-1000ms arrival spacing, and 1-30 servers")
	}
	states := make([]State, len(c.Servers))
	for i, server := range c.Servers {
		if server.ServiceMS < 50 || server.ServiceMS > 5000 {
			return Result{}, fmt.Errorf("service time must be 50–5000ms")
		}
		server.Name = string(rune('A' + i))
		if i >= 26 {
			server.Name = fmt.Sprintf("S%d", i+1)
		}
		states[i].Server = server
	}
	result := Result{Events: []Event{}}
	jobs := []pending{}
	next, rejected := 0, 0
	add := func(at, request int, kind, name, reason string) {
		snapshot := append([]State(nil), states...)
		result.Events = append(result.Events, Event{at, request, kind, name, reason, snapshot, rejected})
	}
	completeUntil := func(until int) {
		sort.SliceStable(jobs, func(i, j int) bool { return jobs[i].end < jobs[j].end })
		for len(jobs) > 0 && jobs[0].end <= until {
			job := jobs[0]
			jobs = jobs[1:]
			states[job.server].Active--
			states[job.server].Completed++
			add(job.end, job.request, "completed", states[job.server].Name, "Work finished; one active connection is released.")
		}
	}
	for request := 1; request <= c.Users; request++ {
		at := (request - 1) * c.IntervalMS
		completeUntil(at)
		chosen := -1
		for offset := 0; offset < len(states); offset++ {
			i := (next + offset) % len(states)
			if !states[i].Healthy {
				continue
			}
			if chosen == -1 || c.Algorithm == "least_connections" && states[i].Active < states[chosen].Active {
				chosen = i
			}
			if c.Algorithm == "round_robin" {
				break
			}
		}
		if chosen == -1 {
			rejected++
			add(at, request, "rejected", "-", "No known-healthy servers. Reject this request (modeled 503).")
			continue
		}
		reason := "Next known-healthy server in rotation; speed is not considered."
		if c.Algorithm == "least_connections" {
			reason = fmt.Sprintf("Fewest active connections: %d before assignment. Ties rotate.", states[chosen].Active)
		}
		next = (chosen + 1) % len(states)
		states[chosen].Active++
		states[chosen].Assigned++
		jobs = append(jobs, pending{at + states[chosen].ServiceMS, request, chosen})
		add(at, request, "routed", states[chosen].Name, reason)
	}
	completeUntil(int(^uint(0) >> 1))
	return result, nil
}
