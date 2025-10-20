package main

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"network-tunneler/internal/agent"
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
		Use:   "agent",
		Short: "Network tunneler agent - intercepts and forwards traffic to server",
		Run:   run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file (yaml/json/.env)")
	rootCmd.Flags().StringVar(&serverAddr, "server", "", "Server address (overrides config)")
	rootCmd.Flags().StringVar(&targetCIDR, "cidr", "", "Target CIDR to intercept (overrides config)")
	rootCmd.Flags().IntVar(&listenPort, "port", 0, "Local listen port (overrides config)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	app := fx.New(
		fx.Supply(configFile),
		fx.Decorate(applyOverrides),

		logger.Module,
		agent.Module,

		fx.Invoke(func(*agent.Agent) {}),
	)

	app.Run()
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
