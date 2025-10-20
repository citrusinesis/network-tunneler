package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"network-tunneler/internal/certs"
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
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          caPool,
		RootCAs:            caPool,
	}

	return tlsConfig, nil
}

func loadCertificate(cfg *config.TLSConfig) (tls.Certificate, error) {
	if cfg.CertPath != "" && cfg.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("failed to load server certificate: %w", err)
		}
		return cert, nil
	}

	cert, err := tls.X509KeyPair([]byte(certs.ServerCert), []byte(certs.ServerKey))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load embedded server certificate: %w", err)
	}
	return cert, nil
}

func loadCAPool(cfg *config.TLSConfig) (*x509.CertPool, error) {
	var caPEM []byte
	var err error

	if cfg.CAPath != "" {
		caPEM, err = os.ReadFile(cfg.CAPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
	} else {
		caPEM = []byte(certs.CACert)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return caPool, nil
}
