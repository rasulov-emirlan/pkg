package router

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
)

type Multiplexer struct {
	ctxCancel context.CancelFunc
	ctx       context.Context
	address   string
	handlers  map[string]Handler
}

type Handler func(req Request, resp *Response) error

func NewMultiplexer(address string) *Multiplexer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Multiplexer{
		ctxCancel: cancel,
		ctx:       ctx,
		address:   address,
		handlers:  make(map[string]Handler),
	}
}

func (m Multiplexer) ListenAndServe() error {
	listener, err := net.Listen("tcp", m.address)
	if err != nil {
		return err
	}
	for {
		select {
		case <-m.ctx.Done():
			return listener.Close()
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && !ne.Timeout() {
				return err
			}
			continue
		}

		go m.Handle(conn)
	}
}

func (m Multiplexer) Shutdown() error {
	m.ctxCancel()
	return nil
}

func (m *Multiplexer) HandleFunc(path string, handler Handler) {
	m.handlers[path] = handler
}

func (m *Multiplexer) Handle(conn net.Conn) {
	defer (conn).Close()
	// read request
	req, err := parseRequest(conn)
	if err != nil {
		slog.Error("parse request error", slog.String("err", err.Error()))
		return
	}

	for k, v := range req.Headers {
		slog.Info("header", slog.String("key", k), slog.String("value", v))
	}

	resp := Response{
		Ctx:     req.Ctx,
		Headers: make(map[string]string),
		Body:    []byte(""),
	}

	handler, found := m.handlers[req.URL.Path]
	if !found {
		slog.Warn("handler not found, will use default", slog.Any("path", req.URL.Path))
		err = defaultHandler(req, &resp)
		if err != nil {
			slog.Error("default handler error", slog.String("err", err.Error()))
			return
		}
	} else {
		err = handler(req, &resp)
		if err != nil {
			slog.Error("handler error", slog.String("err", err.Error()))
			return
		}
	}

	if resp.Headers["Status"] == "" {
		resp.Headers["Status"] = "200 OK"
	}

	isBodyRead := req.Body.isRead == 1
	if !isBodyRead {
		slog.Debug("body is not read, will read it")
		defer func() {
			if err := req.Body.Close(); err != nil {
				slog.Error("close body error", slog.String("err", err.Error()))
				resp.Headers["Status"] = "500 Internal Server Error"
			}
		}()
	}

	slog.Info("response", slog.String("status", resp.Headers["Status"]), slog.String("body", string(resp.Body)))
	n, err := conn.Write([]byte(fmt.Sprintf(
		"HTTP/1.1 %s\r\n"+
			"Content-Length: %d\r\n"+
			"Content-Type: %s\r\n"+
			"\r\n"+
			"%s",
		resp.Headers["Status"],
		len(resp.Body),
		resp.Headers["Content-Type"],
		resp.Body,
	)))
	if err != nil {
		slog.Error("write response error", slog.String("err", err.Error()))
		return
	}

	if n < len(resp.Body) {
		slog.Error("write response error", slog.String("err", "not all bytes are written"), slog.Group("bytes", slog.Int("n", n), slog.Int("len", len(resp.Body))))
		return
	}
}

func parseRequest(conn net.Conn) (Request, error) {
	res := Request{Ctx: context.Background()}

	reader := bufio.NewReader(conn)
	buff, err := reader.ReadBytes('\n')
	if err != nil {
		return res, fmt.Errorf("read request line error: %v", err)
	}

	// parse request line
	parts := bytes.SplitN(buff, []byte(" "), 3)
	if len(parts) != 3 {
		return res, errors.New("invalid request line")
	}

	res.Method = string(parts[0])
	res.URL, err = url.Parse(string(parts[1]))
	if err != nil {
		return res, fmt.Errorf("parse url error: %v", err)
	}
	res.URL.Scheme = "http"
	res.URL.Host = string(parts[2])

	// parse headers
	res.Headers = make(map[string]string)
	for {
		buff, err := reader.ReadBytes('\n')
		if err != nil {
			return res, fmt.Errorf("read header error: %v", err)
		}
		if len(buff) <= 2 {
			break
		}
		parts := bytes.SplitN(buff, []byte(":"), 2)
		if len(parts) != 2 {
			return res, fmt.Errorf("invalid header: %s", buff)
		}
		key := string(bytes.TrimSpace(parts[0]))
		value := string(bytes.TrimSpace(parts[1]))
		res.Headers[key] = value
	}

	res.Body = NewBody(reader, conn.Close)
	return res, nil
}

func defaultHandler(req Request, resp *Response) error {
	resp.Headers["Content-Length"] = fmt.Sprintf("%d", len(resp.Body))
	resp.Headers["Content-Type"] = "text/json"
	resp.Headers["Status"] = "404 Not Found"

	resp.Body = []byte(`{"error":"not found"}`)
	return nil
}
