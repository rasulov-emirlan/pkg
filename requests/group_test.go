package requests

import (
	"context"
	"math/rand"
	"testing"
	"time"
)

func TestRequestGroup(t *testing.T) {
	reqs := map[string]ReqFunc{
		"one": func(ctx context.Context, req any) (any, error) {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			return 1, nil
		},
		"two": func(ctx context.Context, req any) (any, error) {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			return 2, nil
		},
		"three": func(ctx context.Context, req any) (any, error) {
			time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			return 3, nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	results, err := RequestGroup(ctx, reqs)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != len(reqs) {
		t.Fatalf("expected %d results, got %d", len(reqs), len(results))
	}

	for name, result := range results {
		if _, ok := reqs[name]; !ok {
			t.Fatalf("unexpected result: %s", name)
		}

		if _, ok := result.(int); !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
	}
}
