package crypto

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

type TLSOptions struct {
	CertPath string `mapstructure:"cert_path" json:"cert_path" yaml:"cert_path"`
	KeyPath  string `mapstructure:"key_path" json:"key_path" yaml:"key_path"`
	CAPath   string `mapstructure:"ca_path" json:"ca_path" yaml:"ca_path"`

	// Embedded/inline certificates (used as fallback)
	CertPEM []byte `mapstructure:"cert_pem" json:"cert_pem,omitempty" yaml:"cert_pem,omitempty"`
	KeyPEM  []byte `mapstructure:"key_pem" json:"key_pem,omitempty" yaml:"key_pem,omitempty"`
	CAPEM   []byte `mapstructure:"ca_pem" json:"ca_pem,omitempty" yaml:"ca_pem,omitempty"`

	ServerName         string `mapstructure:"server_name" json:"server_name" yaml:"server_name"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

func LoadTLSConfig(opts TLSOptions) (*tls.Config, error) {
	var cert tls.Certificate
	var err error

	if opts.CertPath != "" && opts.KeyPath != "" {
		cert, err = tls.LoadX509KeyPair(opts.CertPath, opts.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate from files: %w", err)
		}
	} else if len(opts.CertPEM) > 0 && len(opts.KeyPEM) > 0 {
		cert, err = tls.X509KeyPair(opts.CertPEM, opts.KeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to load embedded certificate: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no certificate provided (need either file paths or PEM data)")
	}

	var caCertPEM []byte
	if opts.CAPath != "" {
		var err error
		caCertPEM, err = readFile(opts.CAPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA from file: %w", err)
		}
	} else if len(opts.CAPEM) > 0 {
		caCertPEM = opts.CAPEM
	} else {
		return nil, fmt.Errorf("no CA certificate provided (need either file path or PEM data)")
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
		InsecureSkipVerify: opts.InsecureSkipVerify,
	}

	if opts.ServerName != "" {
		tlsConfig.ServerName = opts.ServerName
	}

	return tlsConfig, nil
}

func LoadServerTLSConfig(opts TLSOptions) (*tls.Config, error) {
	tlsConfig, err := LoadTLSConfig(opts)
	if err != nil {
		return nil, err
	}

	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	return tlsConfig, nil
}

func LoadClientTLSConfig(opts TLSOptions) (*tls.Config, error) {
	return LoadTLSConfig(opts)
}
