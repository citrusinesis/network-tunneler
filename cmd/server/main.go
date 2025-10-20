package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"network-tunneler/internal/server"
	"network-tunneler/internal/version"
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
		Use:     "server",
		Short:   "Network tunneler server - central relay for agents and implants",
		Version: version.Short(),
		Run:     run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file (yaml/json/.env)")
	rootCmd.Flags().StringVar(&certPath, "cert", "", "TLS certificate file (overrides config and embedded cert)")
	rootCmd.Flags().StringVar(&keyPath, "key", "", "TLS key file (overrides config and embedded key)")
	rootCmd.Flags().StringVar(&caPath, "ca", "", "CA certificate file (overrides config and embedded CA)")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				info := version.Get()
				data, _ := json.MarshalIndent(info, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Println(version.String())
			}
		},
	}
	versionCmd.Flags().Bool("json", false, "Output version info as JSON")
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	var log logger.Logger

	app := fx.New(
		fx.Supply(configFile),
		fx.Decorate(applyTLSOverrides),

		logger.Module,
		server.Module,

		fx.WithLogger(logger.NewFxLogger),

		fx.Populate(&log),
		fx.Invoke(func(*server.Server) {}),
	)

	if err := app.Start(cmd.Context()); err != nil {
		if log != nil {
			log.Error("failed to start server", logger.Error(err))
		} else {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		}
		os.Exit(1)
	}

	<-app.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), fx.DefaultTimeout)
	defer cancel()

	if err := app.Stop(shutdownCtx); err != nil {
		log.Error("error during shutdown", logger.Error(err))
		os.Exit(1)
	}

	log.Info("server shutdown complete")
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
