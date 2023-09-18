package main

import (
	"log/slog"
	"os"
	"os/signal"

	"github.com/rasulov-emirlan/pkg/router"
)

func main() {
	mux := router.NewMultiplexer(":8080")

	mux.HandleFunc("/", func(req router.Request, resp *router.Response) error {
		resp.Headers["Content-Type"] = "text/html"
		resp.Body = []byte("<h1>Hello, World!</h1>")
		return nil
	})

	mux.HandleFunc("/about", func(req router.Request, resp *router.Response) error {
		resp.Headers["Content-Type"] = "text/html"
		resp.Headers["Status"] = "200 OK"
		resp.Body = []byte("<h1>About</h1>")
		return nil
	})

	go func() {
		slog.Error("listen", "error", mux.ListenAndServe())
	}()

	slog.Info("server", "started", "true")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	err := mux.Shutdown()
	if err != nil {
		slog.Error("shutdown", "error", err)
	}
	slog.Info("server", "stopped", "true")
}
