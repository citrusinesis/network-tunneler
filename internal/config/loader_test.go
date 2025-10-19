package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderDotEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	envContent := `TEST_VALUE=hello
TEST_NUMBER=42
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	loader := NewLoader("test")
	if err := loader.LoadEnvFile(envPath); err != nil {
		t.Fatalf("failed to load .env: %v", err)
	}

	if val := os.Getenv("TEST_VALUE"); val != "hello" {
		t.Errorf("expected env var TEST_VALUE=hello, got %s", val)
	}

	if val := loader.Get("value"); val != "hello" {
		t.Errorf("expected viper to get 'value' as 'hello', got %v", val)
	}
}
