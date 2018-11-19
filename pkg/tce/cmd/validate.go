package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/aspenmesh/tce/pkg/tce/webhook"
	"github.com/aspenmesh/tce/pkg/trafficclaim"
)

func startValidateServer(port int, keypair tls.Certificate) (func(), error) {
	var server *http.Server
	shutdown := func() {
		if server != nil {
			server.Shutdown(context.Background())
		}
	}

	kubeclient, err := trafficclaim.CreateInterface("")
	if err != nil {
		return shutdown, fmt.Errorf("Failed to create a kubernetes client: %v", err)
	}
	claimDb := trafficclaim.NewDb(kubeclient)

	mux := http.NewServeMux()
	if _, err := webhook.NewWebhookServer(mux, claimDb); err != nil {
		return shutdown, fmt.Errorf("Failed to create webhook server: %v", err)

	}
	server = &http.Server{
		Addr:      fmt.Sprintf(":%v", port),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{keypair}},
		Handler:   mux,
	}
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return shutdown, err
	}
	go server.ServeTLS(listener, "", "")
	return shutdown, nil
}
