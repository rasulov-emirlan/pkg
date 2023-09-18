package router

import "context"

type Response struct {
	Ctx     context.Context
	Headers map[string]string
	Body    []byte
}
