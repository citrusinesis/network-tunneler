package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"network-tunneler/internal/certs"
	"network-tunneler/internal/config"
	"network-tunneler/pkg/logger"
)

func LoadTLSConfig(cfg *config.TLSConfig, log logger.Logger) (*tls.Config, error) {
	log = log.With(logger.String("component", "tls"))

	cert, err := loadCertificate(cfg, log)
	if err != nil {
		return nil, err
	}

	caPool, err := loadCAPool(cfg, log)
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

func loadCertificate(cfg *config.TLSConfig, log logger.Logger) (tls.Certificate, error) {
	if cfg.CertPath != "" && cfg.KeyPath != "" {
		log.Debug("loading TLS certificate from files",
			logger.String("cert", cfg.CertPath),
			logger.String("key", cfg.KeyPath),
		)
		cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("failed to load server certificate: %w", err)
		}
		return cert, nil
	}

	log.Debug("loading TLS certificate from embedded data")
	cert, err := tls.X509KeyPair([]byte(certs.ServerCert), []byte(certs.ServerKey))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load embedded server certificate: %w", err)
	}
	return cert, nil
}

func loadCAPool(cfg *config.TLSConfig, log logger.Logger) (*x509.CertPool, error) {
	var caPEM []byte
	var err error

	if cfg.CAPath != "" {
		log.Debug("loading CA certificate from file",
			logger.String("ca", cfg.CAPath),
		)
		caPEM, err = os.ReadFile(cfg.CAPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
	} else {
		log.Debug("loading CA certificate from embedded data")
		caPEM = []byte(certs.CACert)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return caPool, nil
}
