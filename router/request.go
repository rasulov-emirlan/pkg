package router

import (
	"context"
	"net/url"
)

type Request struct {
	Ctx        context.Context
	Headers    map[string]string
	Method     string
	URL        *url.URL
	Host       string
	Path       string
	Proto      string // "HTTP/1.0"
	ProtoMajor int    // 1
	ProtoMinor int    // 0
	Body       []byte
}
