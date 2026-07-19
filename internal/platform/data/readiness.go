package data

import (
	"context"
	"sync"
	"time"
)

const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
	StatusNotReady = "not_ready"

	DependencyStatusOK          = "ok"
	DependencyStatusUnavailable = "unavailable"
)

type Readiness interface {
	Check(context.Context) Status
}

type Status struct {
	Status string `json:"status"`
	MySQL  string `json:"mysql"`
	Redis  string `json:"redis"`
}

type Probe func(context.Context) error

type Checker struct {
	mysqlProbe Probe
	redisProbe Probe
	timeout    time.Duration
}

func NewChecker(mysqlProbe, redisProbe Probe) *Checker {
	return newCheckerWithTimeout(mysqlProbe, redisProbe, dependencyProbeTimeout)
}

func newCheckerWithTimeout(mysqlProbe, redisProbe Probe, timeout time.Duration) *Checker {
	return &Checker{mysqlProbe: mysqlProbe, redisProbe: redisProbe, timeout: timeout}
}

func NewReadiness(clients *Clients) Readiness {
	return NewChecker(
		clients.MySQL.PingContext,
		func(ctx context.Context) error { return clients.Redis.Ping(ctx).Err() },
	)
}

func (c *Checker) Check(ctx context.Context) Status {
	probeCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	mysqlStatus := DependencyStatusOK
	redisStatus := DependencyStatusOK
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := c.mysqlProbe(probeCtx); err != nil {
			mysqlStatus = DependencyStatusUnavailable
		}
	}()
	go func() {
		defer wg.Done()
		if err := c.redisProbe(probeCtx); err != nil {
			redisStatus = DependencyStatusUnavailable
		}
	}()
	wg.Wait()

	overall := StatusOK
	if mysqlStatus == DependencyStatusUnavailable {
		overall = StatusNotReady
	} else if redisStatus == DependencyStatusUnavailable {
		overall = StatusDegraded
	}

	return Status{Status: overall, MySQL: mysqlStatus, Redis: redisStatus}
}
