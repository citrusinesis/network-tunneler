package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"network-tunneler/internal/config"
)

func LoadTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	cert, err := loadCertificate(cfg)
	if err != nil {
		return nil, err
	}

	caPool, err := loadCAPool(cfg)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		ServerName:   "server",
		RootCAs:      caPool,
	}

	return tlsConfig, nil
}

func loadCertificate(cfg *config.TLSConfig) (tls.Certificate, error) {
	if cfg.CertPath == "" || cfg.KeyPath == "" {
		return tls.Certificate{}, fmt.Errorf("cert_path and key_path are required")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load certificate: %w", err)
	}

	return cert, nil
}

func loadCAPool(cfg *config.TLSConfig) (*x509.CertPool, error) {
	if cfg.CAPath == "" {
		return nil, fmt.Errorf("ca_path is required")
	}

	caPEM, err := os.ReadFile(cfg.CAPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return caPool, nil
}
