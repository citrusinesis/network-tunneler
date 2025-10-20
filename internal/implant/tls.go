package implant

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"network-tunneler/internal/config"
	"network-tunneler/pkg/logger"
)

func LoadTLSConfig(cfg *config.TLSConfig, log logger.Logger) (*tls.Config, error) {
	cert, err := loadCertificate(cfg, log)
	if err != nil {
		return nil, err
	}

	caPool, err := loadCAPool(cfg, log)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		ServerName:   "server",
		RootCAs:      caPool,
	}

	log.Info("TLS configuration loaded")

	return tlsConfig, nil
}

func loadCertificate(cfg *config.TLSConfig, log logger.Logger) (tls.Certificate, error) {
	if cfg.CertPath == "" || cfg.KeyPath == "" {
		return tls.Certificate{}, fmt.Errorf("cert_path and key_path are required")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load certificate: %w", err)
	}

	log.Info("certificate loaded",
		logger.String("cert_path", cfg.CertPath),
		logger.String("key_path", cfg.KeyPath),
	)

	return cert, nil
}

func loadCAPool(cfg *config.TLSConfig, log logger.Logger) (*x509.CertPool, error) {
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

	log.Info("CA certificate loaded",
		logger.String("ca_path", cfg.CAPath),
	)

	return caPool, nil
}
