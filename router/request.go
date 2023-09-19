package router

import (
	"context"
	"io"
	"net/url"
	"sync/atomic"
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
	Body       *Body  // TODO: make io.ReadCloser
}

type Body struct {
	io.Reader
	closeFunc func() error
	isRead    uint32 // 1 if read, 0 otherwise
}

func NewBody(reader io.Reader, closeFunc func() error) *Body {
	if closeFunc == nil {
		panic("closeFunc is nil")
	}
	if reader == nil {
		panic("reader is nil")
	}
	return &Body{
		Reader:    reader,
		closeFunc: closeFunc,
		isRead:    0,
	}
}

func (b *Body) Read(p []byte) (n int, err error) {
	atomic.StoreUint32(&b.isRead, 1)
	return b.Reader.Read(p)
}

func (b *Body) Close() error {
	return b.closeFunc()
}
