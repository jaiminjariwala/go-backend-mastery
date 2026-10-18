package loadbalancer

import "testing"

func TestVariableServerCounts(t *testing.T) {
	for _, n := range []int{1, 10, 30} {
		servers := make([]Server, n)
		for i := range servers {
			servers[i] = Server{Healthy: i < 20, ServiceMS: 500}
		}
		result, err := Run(Config{Algorithm: "round_robin", Users: 100, IntervalMS: 100, Servers: servers})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Events) != 200 {
			t.Fatalf("got %d events", len(result.Events))
		}
		final := result.Events[len(result.Events)-1]
		if len(final.States) != n {
			t.Fatal("incorrect server count")
		}
		for _, state := range final.States {
			if !state.Healthy && state.Assigned != 0 {
				t.Fatal("routed to offline server")
			}
		}
	}
}

func TestOverlappingRequestsAreConcurrent(t *testing.T) {
	result, err := Run(Config{Algorithm: "round_robin", Users: 2, IntervalMS: 100, Servers: []Server{{Healthy: true, ServiceMS: 500}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Events[1].States[0].Active != 2 {
		t.Fatal("expected two concurrent requests")
	}
	if result.Events[2].TimeMS != 500 || result.Events[3].TimeMS != 600 {
		t.Fatal("requests should not wait in a queue")
	}
}
