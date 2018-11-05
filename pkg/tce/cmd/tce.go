package cmd

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/aspenmesh/tce/pkg/tce/webhook"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "tce",
	Short: "TrafficClaim Enforcer",
	Long:  "Run the TrafficClaim Enforcer to manage Istio policy interactions",
}

type ServerParms struct {
	port     int
	certFile string
	keyFile  string
}

func Execute() {
	fmt.Println("Execute")

	parms := ServerParms{}

	flag.IntVar(&parms.port, "port", 443, "Webhook server port.")
	flag.StringVar(
		&parms.certFile,
		"tlsCertFile",
		"/etc/webhook/certs/cert.pem",
		"x509 Certificate for HTTPS.",
	)
	flag.StringVar(
		&parms.keyFile,
		"tlsKeyFile",
		"/etc/webhook/certs/key.pem",
		"x509 private key for --tlsCertFile.",
	)
	flag.Parse()

	keypair, err := tls.LoadX509KeyPair(parms.certFile, parms.keyFile)
	if err != nil {
		glog.Fatalf("Failed to load x509 cert/key pair: %v", err)
	}

	mux := http.NewServeMux()
	if _, err := webhook.NewWebhookServer(mux); err != nil {
		glog.Fatalf("Failed to create webhook server: %v", err)

	}
	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", parms.port),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{keypair}},
		Handler:   mux,
	}

	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve: %v", err)
		}
	}()

	waitSignal := make(chan os.Signal, 1)
	signal.Notify(waitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-waitSignal

	glog.Infof("Shutting down")
	if err := server.Shutdown(context.Background()); err != nil {
		glog.Errorf("Error shutting down server, ignored: %v", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
}
