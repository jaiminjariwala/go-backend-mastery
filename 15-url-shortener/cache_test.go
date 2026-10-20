package main

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeInfra struct {
	value   string
	err     error
	allowed bool
	puts    int
}

func (f *fakeInfra) Get(context.Context, string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.value == "" {
		return "", errCacheMiss
	}
	return f.value, nil
}
func (f *fakeInfra) Put(_ context.Context, _, value string) error {
	f.puts++
	f.value = value
	return f.err
}
func (f *fakeInfra) Allow(context.Context, string) (bool, time.Duration, error) {
	return f.allowed, 10 * time.Second, f.err
}
func TestCacheMissThenHitAndBaseline(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	infra := &fakeInfra{}
	app := &App{db: db, infra: infra, cacheEnabled: true}
	mock.ExpectQuery("SELECT original_url").WithArgs("aB3kP9xQ").WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("https://example.com"))
	first, err := app.resolve(context.Background(), "aB3kP9xQ", false)
	if err != nil || first.Cache != "miss" || !first.DatabaseRead || !first.CacheSaved {
		t.Fatal(first, err)
	}
	second, err := app.resolve(context.Background(), "aB3kP9xQ", false)
	if err != nil || second.Source != "redis" || second.DatabaseRead {
		t.Fatal(second, err)
	}
	mock.ExpectQuery("SELECT original_url").WithArgs("aB3kP9xQ").WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("https://example.com"))
	baseline, err := app.resolve(context.Background(), "aB3kP9xQ", true)
	if err != nil || !baseline.DatabaseRead || baseline.CacheSaved {
		t.Fatal(baseline, err)
	}
	if app.metrics.dbReads.Load() != 2 || app.metrics.hits.Load() != 1 {
		t.Fatal("incorrect metrics")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestRedisFailureFallback(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectQuery("SELECT original_url").WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("https://example.com"))
	app := &App{db: db, infra: &fakeInfra{err: errors.New("offline")}, cacheEnabled: true}
	trace, err := app.resolve(context.Background(), "aB3kP9xQ", false)
	if err != nil || !trace.DatabaseRead || trace.Source != "postgres" || trace.CacheSaved {
		t.Fatal(trace, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestRateLimitRejectAndFailure(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{{nil, 429}, {errors.New("private Redis details"), 503}} {
		app := &App{infra: &fakeInfra{err: tt.err}, limitEnabled: true}
		w := httptest.NewRecorder()
		app.Routes().ServeHTTP(w, httptest.NewRequest("GET", "/s/aB3kP9xQ", nil))
		if w.Code != tt.status {
			t.Fatal(w.Code, w.Body.String())
		}
		if tt.status == 429 && w.Header().Get("Retry-After") == "" {
			t.Fatal("missing retry header")
		}
	}
}

func TestInspectAndStats(t *testing.T) {
	app := &App{infra: &fakeInfra{value: "https://example.com", allowed: true}, cacheEnabled: true, limitEnabled: true, instance: "go-test"}
	response := httptest.NewRecorder()
	app.Routes().ServeHTTP(response, httptest.NewRequest("GET", "/api/inspect/aB3kP9xQ", nil))
	if response.Code != 200 || response.Header().Get("X-RateLimit-Decision") != "allowed" {
		t.Fatal(response.Code, response.Body.String())
	}
	if response.Header().Get("Location") != "" {
		t.Fatal("inspect must not redirect")
	}
	if app.metrics.hits.Load() != 1 || app.metrics.dbReads.Load() != 0 {
		t.Fatal("unexpected lookup counters")
	}
	stats := httptest.NewRecorder()
	app.Routes().ServeHTTP(stats, httptest.NewRequest("GET", "/api/stats", nil))
	if stats.Code != 200 {
		t.Fatal(stats.Code)
	}
}
