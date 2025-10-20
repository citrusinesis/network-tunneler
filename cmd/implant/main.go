package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"network-tunneler/internal/implant"
	"network-tunneler/internal/version"
	"network-tunneler/pkg/logger"
)

var (
	configFile string
	serverAddr string
	implantID  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "implant",
		Short:   "Network tunneler implant - forwards traffic to internal network",
		Version: version.Short(),
		Run:     run,
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file (yaml/json/.env)")
	rootCmd.Flags().StringVar(&serverAddr, "server", "", "Server address (overrides config)")
	rootCmd.Flags().StringVar(&implantID, "id", "", "Implant ID (overrides config)")

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
	app := fx.New(
		fx.Supply(configFile),
		fx.Decorate(applyOverrides),

		logger.Module,
		implant.Module,

		fx.Invoke(func(*implant.Implant) {}),
	)

	app.Run()
}

func applyOverrides(cfg *implant.Config) *implant.Config {
	if serverAddr != "" {
		cfg.ServerAddr = serverAddr
	}
	if implantID != "" {
		cfg.ImplantID = implantID
	}
	return cfg
}
