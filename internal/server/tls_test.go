package server

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"network-tunneler/internal/certs"
	"network-tunneler/internal/config"
	testutil "network-tunneler/internal/testing"
)

func TestLoadTLSConfig_Embedded(t *testing.T) {
	cfg := &config.TLSConfig{
		CertPath:           "",
		KeyPath:            "",
		CAPath:             "",
		InsecureSkipVerify: false,
	}

	log := testutil.NewTestLogger()

	tlsConfig, err := LoadTLSConfig(cfg, log)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(tlsConfig.Certificates))
	}

	if tlsConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3, got %d", tlsConfig.MinVersion)
	}

	if tlsConfig.ClientCAs == nil {
		t.Error("expected ClientCAs to be set")
	}
}

func TestLoadTLSConfig_FromFiles(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "test.crt")
	keyPath := filepath.Join(tmpDir, "test.key")
	caPath := filepath.Join(tmpDir, "ca.crt")

	if err := os.WriteFile(certPath, []byte(certs.ServerCert), 0644); err != nil {
		t.Fatalf("failed to write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte(certs.ServerKey), 0600); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}
	if err := os.WriteFile(caPath, []byte(certs.CACert), 0644); err != nil {
		t.Fatalf("failed to write CA: %v", err)
	}

	cfg := &config.TLSConfig{
		CertPath:           certPath,
		KeyPath:            keyPath,
		CAPath:             caPath,
		InsecureSkipVerify: false,
	}

	log := testutil.NewTestLogger()

	tlsConfig, err := LoadTLSConfig(cfg, log)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	if len(tlsConfig.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(tlsConfig.Certificates))
	}
}
