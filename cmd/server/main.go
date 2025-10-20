package main

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"network-tunneler/internal/server"
	"network-tunneler/pkg/logger"
)

var (
	configFile string
	certPath   string
	keyPath    string
	caPath     string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "server",
		Short: "Network tunneler server - central relay for agents and implants",
		Run:   run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file (yaml/json/.env)")
	rootCmd.Flags().StringVar(&certPath, "cert", "", "TLS certificate file (overrides config and embedded cert)")
	rootCmd.Flags().StringVar(&keyPath, "key", "", "TLS key file (overrides config and embedded key)")
	rootCmd.Flags().StringVar(&caPath, "ca", "", "CA certificate file (overrides config and embedded CA)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	app := fx.New(
		fx.Supply(configFile),
		fx.Decorate(applyTLSOverrides),

		logger.Module,
		server.Module,

		fx.Invoke(func(*server.Server) {}),
	)

	app.Run()
}

func applyTLSOverrides(cfg *server.Config) *server.Config {
	if certPath != "" {
		cfg.TLS.CertPath = certPath
	}
	if keyPath != "" {
		cfg.TLS.KeyPath = keyPath
	}
	if caPath != "" {
		cfg.TLS.CAPath = caPath
	}
	return cfg
}
