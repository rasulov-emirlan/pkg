package aggregate

import (
	"context"
	"sync/atomic"
)

type Callback[T any] func(ctx context.Context, args T) error

func CallAfterCount[T any](ctx context.Context, count int64, callbacks ...Callback[T]) Callback[T] {
	var internalCount int64
	return func(ctx context.Context, args T) error {
		atomic.AddInt64(&internalCount, 1)
		if internalCount == count {
			for _, callback := range callbacks {
				if err := callback(ctx, args); err != nil {
					return err
				}
			}
			atomic.StoreInt64(&internalCount, 0)
		}
		return nil
	}
}
