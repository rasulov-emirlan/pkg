package aggregate

import (
	"context"
	"testing"
)

func TestCallAfterCount(t *testing.T) {
	callCount := 0

	callback := CallAfterCount(context.Background(), 10, func(ctx context.Context) error {
		callCount++
		return nil
	})

	for i := 0; i < 1000; i++ {
		if err := callback(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if callCount != 100 {
		t.Fatalf("expected 100 calls, got %d", callCount)
	}
}
