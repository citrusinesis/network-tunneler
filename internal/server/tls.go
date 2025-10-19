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

type TLSManager struct {
	cfg    *config.TLSConfig
	logger logger.Logger
}

func NewTLSManager(cfg *config.TLSConfig, log logger.Logger) *TLSManager {
	return &TLSManager{
		cfg:    cfg,
		logger: log.With(logger.String("component", "tls")),
	}
}

func (t *TLSManager) LoadConfig() (*tls.Config, error) {
	cert, err := t.loadCertificate()
	if err != nil {
		return nil, err
	}

	caPool, err := t.loadCAPool()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: t.cfg.InsecureSkipVerify,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          caPool,
		RootCAs:            caPool,
	}

	return tlsConfig, nil
}

func (t *TLSManager) loadCertificate() (tls.Certificate, error) {
	if t.cfg.CertPath != "" && t.cfg.KeyPath != "" {
		t.logger.Debug("loading TLS certificate from files",
			logger.String("cert", t.cfg.CertPath),
			logger.String("key", t.cfg.KeyPath),
		)
		cert, err := tls.LoadX509KeyPair(t.cfg.CertPath, t.cfg.KeyPath)
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("failed to load server certificate: %w", err)
		}
		return cert, nil
	}

	t.logger.Debug("loading TLS certificate from embedded data")
	cert, err := tls.X509KeyPair([]byte(certs.ServerCert), []byte(certs.ServerKey))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load embedded server certificate: %w", err)
	}
	return cert, nil
}

func (t *TLSManager) loadCAPool() (*x509.CertPool, error) {
	var caPEM []byte
	var err error

	if t.cfg.CAPath != "" {
		t.logger.Debug("loading CA certificate from file",
			logger.String("ca", t.cfg.CAPath),
		)
		caPEM, err = os.ReadFile(t.cfg.CAPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
	} else {
		t.logger.Debug("loading CA certificate from embedded data")
		caPEM = []byte(certs.CACert)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return caPool, nil
}
