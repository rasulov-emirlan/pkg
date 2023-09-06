package aggregate

import (
	"context"
	"sync/atomic"
)

type Callback func(ctx context.Context) error

func CallAfterCount(ctx context.Context, count int64, callbacks ...Callback) Callback {
	var internalCount int64
	return func(ctx context.Context) error {
		atomic.AddInt64(&internalCount, 1)
		if internalCount == count {
			for _, callback := range callbacks {
				if err := callback(ctx); err != nil {
					return err
				}
			}
			atomic.StoreInt64(&internalCount, 0)
		}
		return nil
	}
}
