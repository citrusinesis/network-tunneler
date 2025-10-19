package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Loader struct {
	v      *viper.Viper
	prefix string
}

func NewLoader(appName string) *Loader {
	v := viper.New()
	prefix := strings.ToUpper(appName)
	v.SetEnvPrefix(prefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	return &Loader{
		v:      v,
		prefix: prefix,
	}
}

func (l *Loader) LoadFile(filepath string) error {
	if filepath == "" {
		return nil
	}

	if isEnvFile(filepath) {
		return l.loadEnvFile(filepath)
	}

	return l.loadConfigFile(filepath)
}

func isEnvFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".env") || strings.HasSuffix(base, ".env")
}

func (l *Loader) loadEnvFile(filepath string) error {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil
	}

	envMap, err := godotenv.Read(filepath)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	for key, value := range envMap {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set env var %s: %w", key, err)
		}
	}

	return nil
}

func (l *Loader) loadConfigFile(filepath string) error {
	l.v.SetConfigFile(filepath)
	if err := l.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return nil
}

func (l *Loader) Unmarshal(cfg any) error {
	l.bindEnvs(cfg, "")
	return l.v.Unmarshal(cfg)
}

func (l *Loader) bindEnvs(iface any, prefix string) {
	v := reflect.ValueOf(iface)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}

		var fullPath string
		if prefix == "" {
			fullPath = tag
		} else {
			fullPath = prefix + "." + tag
		}

		l.v.BindEnv(fullPath)

		fieldValue := v.Field(i)
		if fieldValue.Kind() == reflect.Struct {
			l.bindEnvs(fieldValue.Addr().Interface(), fullPath)
		}
	}
}

func (l *Loader) Set(key string, value any) {
	l.v.Set(key, value)
}

func (l *Loader) Get(key string) any {
	return l.v.Get(key)
}
