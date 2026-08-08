package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/amavis442/mockserver/internal/engine"
	"github.com/amavis442/mockserver/internal/server"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	configFile := flag.String("config", "", "path to JSON config file with expectations")
	tlsCert := flag.String("tls-cert", "", "path to TLS certificate (PEM)")
	tlsKey := flag.String("tls-key", "", "path to TLS private key (PEM)")
	tlsSelfSigned := flag.Bool("tls-self-signed", false, "auto-generate a self-signed certificate for localhost")
	flag.Parse()

	store := engine.NewStore()

	if *configFile != "" {
		if err := server.LoadConfig(store, *configFile); err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config %q: %v\n", *configFile, err)
			os.Exit(1)
		}
		log.Printf("loaded %d expectations from %s", len(store.List()), *configFile)
	}

	handler := server.NewHandler(store)
	addr := fmt.Sprintf(":%d", *port)

	var useTLS bool
	var scheme string

	if *tlsSelfSigned {
		certPEM, keyPEM, err := server.GenerateSelfSigned()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to generate self-signed certificate: %v\n", err)
			os.Exit(1)
		}
		// Write to temp files for ListenAndServeTLS (stdlib requires paths).
		certFile, err := writeTempFile("mockserver-cert-*.pem", certPEM)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write temp cert: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(certFile)

		keyFile, err := writeTempFile("mockserver-key-*.pem", keyPEM)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write temp key: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(keyFile)

		*tlsCert = certFile
		*tlsKey = keyFile
		useTLS = true
		scheme = "https"
		log.Printf("using self-signed certificate (valid for localhost)")
	} else if *tlsCert != "" && *tlsKey != "" {
		useTLS = true
		scheme = "https"
		log.Printf("using TLS certificate from %s", *tlsCert)
	} else if *tlsCert != "" || *tlsKey != "" {
		fmt.Fprintln(os.Stderr, "both --tls-cert and --tls-key must be provided together")
		os.Exit(1)
	} else {
		scheme = "http"
	}

	log.Printf("MockServer listening on %s://localhost%s", scheme, addr)
	log.Printf("Admin API at %s://localhost%s/__admin", scheme, addr)

	var listenErr error
	if useTLS {
		listenErr = http.ListenAndServeTLS(addr, *tlsCert, *tlsKey, handler)
	} else {
		listenErr = http.ListenAndServe(addr, handler)
	}

	if listenErr != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", listenErr)
		os.Exit(1)
	}
}

func writeTempFile(pattern string, data []byte) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
