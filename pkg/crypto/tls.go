package crypto

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"network-tunneler/internal/config"
)

func LoadTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	caCert, err := LoadCA(cfg.CAPath, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load CA: %w", err)
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert.Cert)

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		ClientCAs:          caPool,
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	return tlsConfig, nil
}

func LoadServerTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	tlsConfig, err := LoadTLSConfig(cfg)
	if err != nil {
		return nil, err
	}

	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	return tlsConfig, nil
}

func LoadClientTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	tlsConfig, err := LoadTLSConfig(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.ServerName != "" {
		tlsConfig.ServerName = cfg.ServerName
	}

	return tlsConfig, nil
}
