package biz

import (
	"context"
	"time"
)

const (
	defaultReclaimInterval = 30 * time.Second
	defaultReclaimBatch    = 100
)

type ReclaimUseCase interface {
	ReclaimExpired(context.Context, time.Time, int) (int, error)
}

type Reclaimer struct {
	uc       ReclaimUseCase
	now      func() time.Time
	interval time.Duration
	batch    int
	onRun    func(int)
}

func NewReclaimer(uc ReclaimUseCase, now func() time.Time, onRun func(int)) *Reclaimer {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Reclaimer{uc: uc, now: now, interval: defaultReclaimInterval, batch: defaultReclaimBatch, onRun: onRun}
}

func (r *Reclaimer) RunOnce(ctx context.Context) (int, error) {
	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count, err := r.uc.ReclaimExpired(ctx, r.now().UTC(), r.batch)
		if err != nil {
			return total, err
		}
		total += count
		if count < r.batch {
			if r.onRun != nil {
				r.onRun(total)
			}
			return total, nil
		}
	}
}

func (r *Reclaimer) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}
	if _, err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
		return err
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
				return err
			}
		}
	}
}
