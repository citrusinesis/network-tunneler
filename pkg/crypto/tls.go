package crypto

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"network-tunneler/internal/certs"
	"network-tunneler/internal/config"
)

func LoadTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	var cert tls.Certificate
	var err error

	if cfg.CertPath != "" && cfg.KeyPath != "" {
		cert, err = tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
	} else {
		cert, err = tls.X509KeyPair([]byte(certs.AgentCert), []byte(certs.AgentKey))
		if err != nil {
			return nil, fmt.Errorf("failed to load embedded certificate: %w", err)
		}
	}

	var caCertPEM []byte
	if cfg.CAPath != "" {
		caCert, err := LoadCA(cfg.CAPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load CA: %w", err)
		}
		caCertPEM = caCert.Cert.Raw
	} else {
		caCertPEM = []byte(certs.CACert)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

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
