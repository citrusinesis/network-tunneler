package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsEnvFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{".env", true},
		{".env.local", true},
		{".env.production", true},
		{"config.env", true},
		{"test/.env", true},
		{"config.yaml", false},
		{"config.json", false},
		{"settings.toml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isEnvFile(tt.path)
			if result != tt.expected {
				t.Errorf("isEnvFile(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestLoaderEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	envContent := `TEST_VALUE=hello
TEST_NUMBER=42
TEST_BOOL=true
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	loader := NewLoader("test")
	if err := loader.LoadFile(envPath); err != nil {
		t.Fatalf("failed to load .env: %v", err)
	}

	if val := os.Getenv("TEST_VALUE"); val != "hello" {
		t.Errorf("expected env var TEST_VALUE=hello, got %s", val)
	}

	if val := os.Getenv("TEST_NUMBER"); val != "42" {
		t.Errorf("expected env var TEST_NUMBER=42, got %s", val)
	}

	if val := loader.Get("value"); val != "hello" {
		t.Errorf("expected viper to get 'value' as 'hello', got %v", val)
	}
}

func TestLoaderYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
server_addr: "localhost:8080"
port: 9999
enabled: true
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	loader := NewLoader("test")
	if err := loader.LoadFile(yamlPath); err != nil {
		t.Fatalf("failed to load yaml: %v", err)
	}

	if val := loader.Get("server_addr"); val != "localhost:8080" {
		t.Errorf("expected server_addr=localhost:8080, got %v", val)
	}

	if val := loader.Get("port"); val != 9999 {
		t.Errorf("expected port=9999, got %v", val)
	}
}

func TestLoaderJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "server_addr": "json.example.com:8080",
  "port": 5555,
  "enabled": false
}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write json: %v", err)
	}

	loader := NewLoader("test")
	if err := loader.LoadFile(jsonPath); err != nil {
		t.Fatalf("failed to load json: %v", err)
	}

	if val := loader.Get("server_addr"); val != "json.example.com:8080" {
		t.Errorf("expected server_addr=json.example.com:8080, got %v", val)
	}

	if val := loader.Get("port"); val != float64(5555) {
		t.Errorf("expected port=5555, got %v", val)
	}
}

func TestLoaderMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "base.yaml")
	yamlContent := `
server_addr: "yaml.example.com:8080"
port: 1111
timeout: 30
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	envPath := filepath.Join(tmpDir, ".env")
	envContent := `TEST_PORT=2222
TEST_ENABLED=true
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	loader := NewLoader("test")

	if err := loader.LoadFile(yamlPath); err != nil {
		t.Fatalf("failed to load yaml: %v", err)
	}

	if err := loader.LoadFile(envPath); err != nil {
		t.Fatalf("failed to load .env: %v", err)
	}

	if val := loader.Get("server_addr"); val != "yaml.example.com:8080" {
		t.Errorf("expected server_addr from yaml, got %v", val)
	}

	port := loader.Get("port")
	if port != 2222 && port != "2222" {
		t.Errorf("expected port=2222 from env (override), got %v (type %T)", port, port)
	}

	if val := loader.Get("timeout"); val != 30 {
		t.Errorf("expected timeout=30 from yaml, got %v", val)
	}
}

func TestLoaderNonExistentFile(t *testing.T) {
	loader := NewLoader("test")

	err := loader.LoadFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLoaderEmptyPath(t *testing.T) {
	loader := NewLoader("test")

	err := loader.LoadFile("")
	if err != nil {
		t.Errorf("expected no error for empty path, got %v", err)
	}
}
