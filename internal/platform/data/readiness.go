package data

import (
	"context"
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
}

func NewChecker(mysqlProbe, redisProbe Probe) *Checker {
	return &Checker{mysqlProbe: mysqlProbe, redisProbe: redisProbe}
}

func NewReadiness(clients *Clients) Readiness {
	return NewChecker(
		clients.MySQL.PingContext,
		func(ctx context.Context) error { return clients.Redis.Ping(ctx).Err() },
	)
}

func (c *Checker) Check(ctx context.Context) Status {
	probeCtx, cancel := context.WithTimeout(ctx, dependencyProbeTimeout)
	defer cancel()

	mysqlStatus := "ok"
	if err := c.mysqlProbe(probeCtx); err != nil {
		mysqlStatus = "unavailable"
	}
	redisStatus := "ok"
	if err := c.redisProbe(probeCtx); err != nil {
		redisStatus = "unavailable"
	}

	overall := "ok"
	if mysqlStatus == "unavailable" {
		overall = "not_ready"
	} else if redisStatus == "unavailable" {
		overall = "degraded"
	}

	return Status{Status: overall, MySQL: mysqlStatus, Redis: redisStatus}
}
