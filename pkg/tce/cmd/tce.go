package cmd

import (
	"crypto/tls"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "tce",
	Short: "TrafficClaim Enforcer",
	Long:  "Run the TrafficClaim Enforcer to manage Istio policy interactions",
}

type ServerParms struct {
	port        int
	metricsPort int
	certFile    string
	keyFile     string
}

func Execute() {
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
	flag.IntVar(
		&parms.metricsPort,
		"metrics",
		9093,
		"Prometheus monitoring port",
	)
	flag.Parse()

	shutdownMetrics, err := startMetricsServer(parms.metricsPort)
	if err != nil {
		glog.Fatalf("Failed to start metrics server: %v", err)
	}
	glog.Infof("Metrics listen addr: %v\n", parms.metricsPort)

	keypair, err := tls.LoadX509KeyPair(parms.certFile, parms.keyFile)
	if err != nil {
		glog.Fatalf("Failed to load x509 cert/key pair: %v", err)
	}
	shutdownValidate, err := startValidateServer(parms.port, keypair)
	if err != nil {
		glog.Fatalf("Failed to start validate server: %v", err)
	}
	glog.Infof("Validate listen addr: %v\n", parms.port)

	waitSignal := make(chan os.Signal, 1)
	signal.Notify(waitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-waitSignal

	glog.Infof("Shutting down")
	shutdownMetrics()
	shutdownValidate()
}
