package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"network-tunneler/internal/proxy"
	"network-tunneler/internal/version"
	"network-tunneler/pkg/logger"
)

var (
	configFile string
	serverAddr string
	proxyID  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "proxy",
		Short:   "Network tunneler proxy - forwards traffic to internal network",
		Version: version.Short(),
		Run:     run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file (yaml/json/.env)")
	rootCmd.Flags().StringVar(&serverAddr, "server", "", "Server address (overrides config)")
	rootCmd.Flags().StringVar(&proxyID, "id", "", "Proxy ID (overrides config)")

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
		fx.Decorate(applyOverrides),

		logger.Module,
		proxy.Module,

		fx.WithLogger(logger.NewFxLogger),

		fx.Populate(&log),
		fx.Invoke(func(*proxy.Proxy) {}),
	)

	if err := app.Start(cmd.Context()); err != nil {
		if log != nil {
			log.Error("failed to start proxy", logger.Error(err))
		} else {
			fmt.Fprintf(os.Stderr, "Failed to start proxy: %v\n", err)
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

	log.Info("proxy shutdown complete")
}

func applyOverrides(cfg *proxy.Config) *proxy.Config {
	if serverAddr != "" {
		cfg.ServerAddr = serverAddr
	}
	if proxyID != "" {
		cfg.ProxyID = proxyID
	}
	return cfg
}
