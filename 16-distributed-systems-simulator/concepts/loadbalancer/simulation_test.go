package loadbalancer

import "testing"

func config() Config {
	return Config{"round_robin", 12, 100, []Server{{"A", true, 500}, {"B", true, 500}, {"C", true, 500}}}
}
func TestRoundRobin(t *testing.T) {
	r, err := Run(config())
	if err != nil {
		t.Fatal(err)
	}
	last := r.Events[len(r.Events)-1]
	for _, s := range last.States {
		if s.Assigned != 4 || s.Completed != 4 || s.Active != 0 {
			t.Error(s)
		}
	}
	for _, e := range r.Events {
		for _, s := range e.States {
			if s.Active < 0 || s.Assigned != s.Active+s.Completed {
				t.Fatal("invalid state", s)
			}
		}
	}
}
func TestHealth(t *testing.T) {
	c := config()
	c.Servers[1].Healthy = false
	r, _ := Run(c)
	for _, e := range r.Events {
		if e.Server == "B" {
			t.Error("unhealthy server used")
		}
	}
	for i := range c.Servers {
		c.Servers[i].Healthy = false
	}
	r, _ = Run(c)
	if len(r.Events) != 12 || r.Events[11].Rejected != 12 {
		t.Error("expected all rejected")
	}
}
func TestSlowServer(t *testing.T) {
	c := config()
	c.Users = 100
	c.Servers[1].ServiceMS = 2000
	r, _ := Run(c)
	rr := r.Events[len(r.Events)-1].States[1].Assigned
	c.Algorithm = "least_connections"
	r, _ = Run(c)
	lc := r.Events[len(r.Events)-1].States[1].Assigned
	if lc >= rr {
		t.Errorf("slow B should get fewer requests: LC %d RR %d", lc, rr)
	}
}
func TestInvalidConfig(t *testing.T) {
	c := config()
	c.Users = 10000
	if _, err := Run(c); err == nil {
		t.Error("unbounded input accepted")
	}
}
