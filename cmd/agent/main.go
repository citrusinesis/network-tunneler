package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"network-tunneler/internal/agent"
	"network-tunneler/internal/version"
	"network-tunneler/pkg/logger"
)

var (
	configFile string
	serverAddr string
	targetCIDR string
	listenPort int
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "agent",
		Short:   "Network tunneler agent - intercepts and forwards traffic to server",
		Version: version.Short(),
		Run:     run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file (yaml/json/.env)")
	rootCmd.Flags().StringVar(&serverAddr, "server", "", "Server address (overrides config)")
	rootCmd.Flags().StringVar(&targetCIDR, "cidr", "", "Target CIDR to intercept (overrides config)")
	rootCmd.Flags().IntVar(&listenPort, "port", 0, "Local listen port (overrides config)")

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
		agent.Module,

		fx.WithLogger(logger.NewFxLogger),

		fx.Populate(&log),
		fx.Invoke(func(*agent.Agent) {}),
	)

	if err := app.Start(cmd.Context()); err != nil {
		if log != nil {
			log.Error("failed to start agent", logger.Error(err))
		} else {
			fmt.Fprintf(os.Stderr, "Failed to start agent: %v\n", err)
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

	log.Info("agent shutdown complete")
}

func applyOverrides(cfg *agent.Config) *agent.Config {
	if serverAddr != "" {
		cfg.ServerAddr = serverAddr
	}
	if targetCIDR != "" {
		cfg.TargetCIDR = targetCIDR
	}
	if listenPort != 0 {
		cfg.ListenPort = listenPort
	}
	return cfg
}
