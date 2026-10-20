package main

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type counters struct{ hits, misses, dbReads, cacheErrors, allowed, rejected, limitErrors atomic.Uint64 }
type lookupTrace struct {
	Code         string  `json:"code"`
	URL          string  `json:"url,omitempty"`
	Source       string  `json:"source"`
	Cache        string  `json:"cache"`
	DatabaseRead bool    `json:"database_read"`
	CacheSaved   bool    `json:"cache_saved"`
	ElapsedMS    float64 `json:"elapsed_ms"`
	Instance     string  `json:"instance"`
}

func (a *App) Stats(w http.ResponseWriter, r *http.Request) {
	reply(w, 200, map[string]any{"instance": a.instance, "redis_configured": a.infra != nil, "cache_enabled": a.infra != nil && a.cacheEnabled, "rate_limit_enabled": a.infra != nil && a.limitEnabled, "cache_ttl_seconds": 60, "limit": 20, "window_seconds": 10, "cache_hits": a.metrics.hits.Load(), "cache_misses": a.metrics.misses.Load(), "lookup_db_reads": a.metrics.dbReads.Load(), "cache_errors": a.metrics.cacheErrors.Load(), "rate_allowed": a.metrics.allowed.Load(), "rate_rejected": a.metrics.rejected.Load(), "rate_errors": a.metrics.limitErrors.Load()})
}
func (a *App) protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Instance-ID", a.instance)
		if a.infra == nil || !a.limitEnabled {
			w.Header().Set("X-RateLimit-Decision", "disabled")
			next(w, r)
			return
		}
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		scope := "lookup:"
		if r.Method == "POST" {
			scope = "create:"
		}
		ctx, cancel := context.WithTimeout(r.Context(), 250*time.Millisecond)
		allowed, retry, err := a.infra.Allow(ctx, scope+ip)
		cancel()
		if err != nil {
			a.metrics.limitErrors.Add(1)
			w.Header().Set("X-RateLimit-Decision", "unavailable")
			fail(w, 503, "Rate limiter unavailable; retry later")
			return
		}
		if !allowed {
			a.metrics.rejected.Add(1)
			w.Header().Set("Retry-After", strconv.Itoa(max(1, int((retry+time.Second-1)/time.Second))))
			w.Header().Set("X-RateLimit-Decision", "rejected")
			fail(w, 429, "Too many requests: 20 per 10 seconds for this client")
			return
		}
		a.metrics.allowed.Add(1)
		w.Header().Set("X-RateLimit-Decision", "allowed")
		next(w, r)
	}
}
func (a *App) resolve(ctx context.Context, code string, bypass bool) (trace lookupTrace, err error) {
	start := time.Now()
	trace = lookupTrace{Code: code, Cache: "disabled", Source: "postgres", Instance: a.instance}
	defer func() { trace.ElapsedMS = float64(time.Since(start).Microseconds()) / 1000 }()
	if a.infra != nil && a.cacheEnabled && !bypass {
		cacheCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		value, cacheErr := a.infra.Get(cacheCtx, code)
		cancel()
		if cacheErr == nil && validURL(value) {
			a.metrics.hits.Add(1)
			trace.URL = value
			trace.Source = "redis"
			trace.Cache = "hit"
			return trace, nil
		}
		if errors.Is(cacheErr, errCacheMiss) {
			a.metrics.misses.Add(1)
			trace.Cache = "miss"
		} else {
			a.metrics.cacheErrors.Add(1)
			trace.Cache = "error; database fallback"
		}
	} else if bypass {
		trace.Cache = "bypassed for baseline"
	}
	trace.DatabaseRead = true
	a.metrics.dbReads.Add(1)
	err = a.db.QueryRowContext(ctx, "SELECT original_url FROM url_shortener.links WHERE code = $1", code).Scan(&trace.URL)
	if err != nil {
		return trace, err
	}
	if !validURL(trace.URL) {
		return trace, errors.New("invalid stored URL")
	}
	if a.infra != nil && a.cacheEnabled && !bypass {
		cacheCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		cacheErr := a.infra.Put(cacheCtx, code, trace.URL)
		cancel()
		trace.CacheSaved = cacheErr == nil
		if cacheErr != nil {
			a.metrics.cacheErrors.Add(1)
		}
	}
	return trace, nil
}
func validCode(code string) bool {
	if len(code) != 8 {
		return false
	}
	for _, c := range code {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func (a *App) Inspect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if !validCode(code) {
		fail(w, 404, "Link not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	trace, err := a.resolve(ctx, code, r.URL.Query().Get("baseline") == "1")
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "Link not found")
		return
	}
	if err != nil {
		fail(w, 500, "Could not look up link")
		return
	}
	reply(w, 200, trace)
}
