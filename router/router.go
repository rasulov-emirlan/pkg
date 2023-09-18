package router

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
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
		slog.Error("parse request error: %v", slog.String("err", err.Error()))
		return
	}

	for k, v := range req.Headers {
		slog.Info("header", slog.String("key", k), slog.String("value", v))
	}

	resp := Response{
		Ctx:     req.Ctx,
		Headers: make(map[string]string),
		Body:    []byte("Hello World"),
	}

	handler, found := m.handlers[req.URL.Path]
	if !found {
		slog.Warn("handler not found, will use default", slog.Any("path", req.URL))
		err = defaultHandler(req, &resp)
		if err != nil {
			slog.Error("default handler error: %v", slog.String("err", err.Error()))
			return
		}
	} else {
		err = handler(req, &resp)
		if err != nil {
			slog.Error("handler error: %v", slog.String("err", err.Error()))
			return
		}
	}

	if resp.Headers["Status"] == "" {
		resp.Headers["Status"] = "200 OK"
	}

	conn.Write([]byte(fmt.Sprintf(
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
}

func parseRequest(conn net.Conn) (Request, error) {
	res := Request{Ctx: context.Background()}

	buff := make([]byte, 10000)

	n, err := (conn).Read(buff)
	if err != nil {
		return res, err
	}

	if n == 0 {
		return res, errors.New("empty request")
	}

	splits := bytes.Split(buff, []byte("\r\n"))
	if len(splits) < 2 {
		return res, errors.New("invalid request")
	}

	// Parse Method and Protocol version
	{
		sp := bytes.Split(splits[0], []byte(" "))
		if len(sp) < 3 {
			return res, errors.New("invalid request: method or protocol version not found")
		}

		res.Method = string(sp[0])
		res.Path = string(sp[1])
		res.Proto = string(sp[2])
		res.ProtoMajor, res.ProtoMinor = 1, 1 // TODO: change to parsing
	}

	// Parse URL
	{
		sp := bytes.Split(splits[1], []byte(" "))
		if len(sp) < 2 {
			return res, errors.New("invalid request: url not found")
		}

		u, err := url.Parse(string(sp[1]))
		if err != nil {
			return res, err
		}

		u.Path = res.Path
		res.URL = u
		res.Host = u.Host
	}

	// Parse Headers
	i := 1
	{
		res.Headers = make(map[string]string)
		for ; i < len(splits); i++ {
			sp := bytes.Split(splits[i], []byte(":"))
			if len(sp) < 2 {
				slog.Info("invalid header", "header", sp)
				continue
			}

			key := strings.TrimSpace(string(sp[0]))
			value := strings.TrimSpace(string(sp[1]))

			res.Headers[key] = value

			if key == "Content-Length" {
				slog.Info("content length", "length", value)
				break
			}
		}
	}

	// Parse Body
	{
		if i < len(splits) {
			contentLen, ok := res.Headers["Content-Length"]
			if contentLen == "" || !ok {
				return res, errors.New("invalid request: content length not found")
			}

			contentLenInt, err := strconv.Atoi(contentLen)
			if err != nil {
				return res, fmt.Errorf("invalid request: content length is not integer: %v", err)
			}

			if len(splits) < i+2 {
				return res, errors.New("invalid request: body not found")
			}

			if len(splits[i+2]) < contentLenInt {
				return res, errors.New("invalid request: body length is not equal to content length")
			}

			res.Body = splits[i+2][:contentLenInt]
		}
	}

	return res, nil
}

func defaultHandler(req Request, resp *Response) error {
	const msg = `
		{
			"method": %q,
			"url": %q,
			"headers": %q,
			"body": %q
		}
	`
	resp.Headers["Content-Length"] = fmt.Sprintf("%d", len(resp.Body))
	resp.Headers["Content-Type"] = "text/json"
	resp.Headers["Status"] = "200 OK"

	headers := ""
	for k, v := range req.Headers {
		headers += fmt.Sprintf("%q=%q,", k, v)
	}

	resp.Body = []byte(fmt.Sprintf(msg, req.Method, req.URL, headers, req.Body))
	return nil
}
