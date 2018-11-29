package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func startMetricsServer(port int) (func(), error) {
	var server *http.Server
	shutdown := func() {
		if server != nil {
			server.Shutdown(context.Background())
		}
	}
	if port == 0 {
		return shutdown, nil
	}

	mux := http.NewServeMux()
	server = &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
	}
	mux.Handle("/metrics", promhttp.Handler())
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return shutdown, err
	}
	go server.Serve(listener)
	return shutdown, nil
}
