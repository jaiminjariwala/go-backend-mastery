// A bounded local load generator. It never follows redirects to external sites.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	target := flag.String("url", "http://127.0.0.1:8081/s/REPLACE1", "local short-link URL")
	count := flag.Int("n", 200, "total requests (1-10000)")
	workers := flag.Int("c", 10, "concurrency (1-100)")
	flag.Parse()
	u, err := url.Parse(*target)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || !strings.HasPrefix(u.Path, "/s/") || *count < 1 || *count > 10000 || *workers < 1 || *workers > 100 {
		panic("use a local http://127.0.0.1 short URL, n 1-10000 and c 1-100")
	}
	transport := &http.Transport{MaxIdleConns: 100, MaxIdleConnsPerHost: 100, Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 6 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	jobs := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	latencies := []float64{}
	statuses := map[int]int{}
	failures := 0
	start := time.Now()
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				at := time.Now()
				r, err := client.Get(*target)
				status := 0
				if err == nil {
					status = r.StatusCode
					io.Copy(io.Discard, r.Body)
					r.Body.Close()
				}
				ms := float64(time.Since(at).Microseconds()) / 1000
				mu.Lock()
				latencies = append(latencies, ms)
				if err != nil {
					failures++
				} else {
					statuses[status]++
				}
				mu.Unlock()
			}
		}()
	}
	for i := 0; i < *count; i++ {
		jobs <- struct{}{}
	}
	close(jobs)
	wg.Wait()
	elapsed := time.Since(start)
	sort.Float64s(latencies)
	successful := statuses[302]
	fmt.Printf("requests=%d concurrency=%d elapsed=%s total_rps=%.2f redirect_rps=%.2f p50_ms=%.3f p95_ms=%.3f transport_errors=%d statuses=%v\n", *count, *workers, elapsed, float64(*count)/elapsed.Seconds(), float64(successful)/elapsed.Seconds(), latencies[len(latencies)/2], latencies[(len(latencies)-1)*95/100], failures, statuses)
	fmt.Println("Latency includes all statuses. A fast 429/500 is not a successful redirect. Record hardware, cache mode, Redis/database location and error counts with results.")
}
