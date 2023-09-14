package requests

import (
	"context"
	"time"
)

type ReqFunc func(ctx context.Context, req any) (any, error)

type response struct {
	name     string
	data     any
	duration time.Duration
	err      error
}

// RequestGroup is a helper function to make multiple requests in parallel.
func RequestGroup(ctx context.Context, reqs map[string]ReqFunc) (map[string]any, error) {
	responses := make(chan response, len(reqs))
	defer close(responses)

	for name, req := range reqs {
		go func(name string, req ReqFunc) {
			start := time.Now()
			data, err := req(ctx, nil)
			responses <- response{
				name:     name,
				data:     data,
				duration: time.Since(start),
				err:      err,
			}
		}(name, req)
	}

	results := make(map[string]any, len(reqs))
	for i := 0; i < len(reqs); i++ {
		res := <-responses
		if res.err != nil {
			return nil, res.err
		}
		results[res.name] = res.data
	}

	return results, nil
}
