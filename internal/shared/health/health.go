// Package health provides dependency-aware health check probes for all Tacito Square components.
// Liveness (/healthz) checks if the process is alive.
// Readiness (/readyz) verifies all architectural dependencies are reachable.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
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
	logger   zerolog.Logger

	mu             sync.Mutex
	previousFailed map[string]bool
}

// NewProbe creates a new health probe with the given dependency checkers.
func NewProbe(timeout time.Duration, checkers ...Checker) *Probe {
	return &Probe{
		checkers:       checkers,
		timeout:        timeout,
		logger:         zerolog.New(os.Stdout).With().Timestamp().Logger(),
		previousFailed: make(map[string]bool),
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
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
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

			p.mu.Lock()
			wasFailed := p.previousFailed[c.Name]
			if err != nil {
				results[idx] = CheckResult{Name: c.Name, Status: "unhealthy", Error: err.Error()}
				allHealthy = false

				// Log only on transition to failed (no noisy logging)
				if !wasFailed {
					p.previousFailed[c.Name] = true
					p.logger.Error().
						Str("component", "health").
						Str("dependency", c.Name).
						Err(err).
						Msg("Dependency transitioned to UNHEALTHY")

					if c.Name == "postgres" && strings.Contains(err.Error(), "server refused TLS connection") {
						p.logger.Warn().
							Msg("DATABASE CONNECTION DIAGNOSIS: The PostgreSQL server refused the TLS connection request. " +
								"This typically means that SSL/TLS is disabled on the server (e.g. 'ssl = off' in postgresql.conf). " +
								"To bypass this in development, modify your connection URL by setting 'sslmode=prefer' or 'sslmode=disable' (e.g., TS_KEEPER_DATABASE_URL=\"postgres://...sslmode=disable\"). " +
								"In production, verify that the PostgreSQL server has TLS enabled and correct certificates are loaded.")
					}
				}
			} else {
				results[idx] = CheckResult{Name: c.Name, Status: "healthy"}

				// Log exactly once on transition back to success
				if wasFailed {
					p.previousFailed[c.Name] = false
					p.logger.Info().
						Str("component", "health").
						Str("dependency", c.Name).
						Msg("Dependency recovered to HEALTHY")
				}
			}
			p.mu.Unlock()
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
	_ = json.NewEncoder(w).Encode(ReadinessResult{
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
