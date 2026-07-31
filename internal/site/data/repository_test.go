package data

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSanitizeDataErrorPreservesContextErrors(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sanitizeDataError(canceledCtx, errors.New("raw driver detail")); !errors.Is(err, context.Canceled) {
		t.Fatalf("sanitizeDataError(canceled context) = %v, want context.Canceled", err)
	}

	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := sanitizeDataError(expiredCtx, errors.New("raw driver detail")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("sanitizeDataError(expired context) = %v, want context.DeadlineExceeded", err)
	}

	if err := sanitizeDataError(context.Background(), context.Canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("sanitizeDataError(raw canceled) = %v, want context.Canceled", err)
	}
	if err := sanitizeDataError(context.Background(), context.DeadlineExceeded); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("sanitizeDataError(raw deadline) = %v, want context.DeadlineExceeded", err)
	}
}

func TestSanitizeDataErrorHidesStorageDetails(t *testing.T) {
	err := sanitizeDataError(context.Background(), errors.New("secret database detail"))
	if !errors.Is(err, errDataUnavailable) {
		t.Fatalf("sanitizeDataError() = %v, want errDataUnavailable", err)
	}
	if strings.Contains(err.Error(), "secret database detail") {
		t.Fatalf("sanitizeDataError() exposed raw storage error: %v", err)
	}
}
