//go:build development

// license-mock is a local development simulator, never a commercial server.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gestor-documental/internal/licensefixture"
	"gestor-documental/internal/licensing"
	"gestor-documental/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Simulator:", err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("use init|serve|issue|vectors --directory PRIVATE_DIR [--listen 127.0.0.1:9443] [--request FILE.licreq --out FILE.lic --key DEMO-PERPETUAL]")
	}
	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	directory := flags.String("directory", "", "Private simulator directory")
	listen := flags.String("listen", "127.0.0.1:9443", "Loopback HTTPS address")
	requestPath := flags.String("request", "", "Offline request")
	output := flags.String("out", "", "Response file or vector directory")
	key := flags.String("key", "DEMO-PERPETUAL", "Public demonstration scenario, never a real key")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unrecognized arguments")
	}
	if command == "vectors" {
		if *output == "" {
			return fmt.Errorf("--out required")
		}
		return licensefixture.GenerateVectors(*output)
	}
	if *directory == "" {
		return fmt.Errorf("--directory required")
	}
	absolute, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	if err = storage.PreparePrivateDirectory(absolute); err != nil {
		return err
	}
	unlock, err := storage.LockState(absolute)
	if err != nil {
		return fmt.Errorf("stop the simulator before processing offline files: %w", err)
	}
	defer unlock()
	cert := filepath.Join(absolute, "server.crt")
	private := filepath.Join(absolute, "server.key")
	if command == "init" {
		if err = createCertificate(cert, private); err != nil {
			return err
		}
		options := licensing.DefaultOptions()
		options.ServerURL = "https://" + *listen
		options.TrustedKeys = []licensing.TrustKey{licensefixture.TrustKey()}
		options.DevelopmentCAFile = cert
		options.DevelopmentBypass = false
		contents, _ := json.MarshalIndent(map[string]any{"license": options}, "", "  ")
		if err = writeNew(filepath.Join(absolute, "license-config.json"), contents); err != nil {
			return err
		}
		fmt.Println("Development simulator prepared. Merge license-config.json into a separate development installation config.")
		return nil
	}
	mock, err := licensefixture.NewMock(filepath.Join(absolute, "mock-state.json"))
	if err != nil {
		return err
	}
	if command == "issue" {
		if *requestPath == "" || *output == "" {
			return fmt.Errorf("--request and --out required")
		}
		contents, err := os.ReadFile(*requestPath)
		if err != nil {
			return err
		}
		response, err := mock.Offline(contents, *key)
		if err != nil {
			return err
		}
		if err = writeNew(*output, append(response, '\n')); err != nil {
			return err
		}
		fmt.Println("Development response created. Deactivation JSON is provider evidence only; it is not an importable license.")
		return nil
	}
	if command != "serve" {
		return fmt.Errorf("unknown command")
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("simulator must listen on an explicit loopback IP")
	}
	server := &http.Server{Addr: *listen, Handler: mock, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}, MaxHeaderBytes: 8192}
	fmt.Println("AIBID development license simulator: https://" + *listen)
	return server.ListenAndServeTLS(cert, private)
}
func writeNew(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = file.Write(contents); err != nil {
		return err
	}
	return file.Sync()
}
func createCertificate(certPath, keyPath string) error {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	template := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "AIBID DEVELOPMENT LOCAL ONLY"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(365 * 24 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true, IsCA: true, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	if err != nil {
		return err
	}
	bytes, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return err
	}
	if err = writeNew(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: bytes})); err != nil {
		return err
	}
	return writeNew(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
