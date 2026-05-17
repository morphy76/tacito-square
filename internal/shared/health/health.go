// Package health provides dependency-aware health check probes for all Tacito Square components.
// Liveness (/healthz) checks if the process is alive.
// Readiness (/readyz) verifies all architectural dependencies are reachable.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Checker is a named dependency health check function.
type Checker struct {
	Name  string
	Check func(ctx context.Context) error
}

// Probe manages liveness and readiness checks.
type Probe struct {
	checkers []Checker
	timeout  time.Duration
}

// NewProbe creates a new health probe with the given dependency checkers.
func NewProbe(timeout time.Duration, checkers ...Checker) *Probe {
	return &Probe{
		checkers: checkers,
		timeout:  timeout,
	}
}

// CheckResult is a single dependency check result.
type CheckResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ReadinessResult is the full readiness check response.
type ReadinessResult struct {
	Status string        `json:"status"`
	Checks []CheckResult `json:"checks"`
}

// LivezHandler handles GET /healthz — always returns 200 if process is alive.
func (p *Probe) LivezHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// ReadyzHandler handles GET /readyz — checks all dependencies in parallel.
func (p *Probe) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), p.timeout)
	defer cancel()

	results := make([]CheckResult, len(p.checkers))
	var wg sync.WaitGroup
	allHealthy := true

	for i, checker := range p.checkers {
		wg.Add(1)
		go func(idx int, c Checker) {
			defer wg.Done()
			err := c.Check(ctx)
			if err != nil {
				results[idx] = CheckResult{Name: c.Name, Status: "unhealthy", Error: err.Error()}
				allHealthy = false
			} else {
				results[idx] = CheckResult{Name: c.Name, Status: "healthy"}
			}
		}(i, checker)
	}

	wg.Wait()

	status := "ready"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(ReadinessResult{
		Status: status,
		Checks: results,
	})
}

// --- Common dependency checkers ---

// PingChecker creates a checker that calls a ping function (e.g., db.Ping, redis.Ping).
func PingChecker(name string, pingFn func(ctx context.Context) error) Checker {
	return Checker{Name: name, Check: pingFn}
}

// HTTPChecker creates a checker that verifies an HTTP endpoint is reachable.
func HTTPChecker(name, url string) Checker {
	return Checker{
		Name: name,
		Check: func(ctx context.Context) error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("returned status %d", resp.StatusCode)
			}
			return nil
		},
	}
}
