package agent

import (
	"fmt"

	"go.uber.org/fx"

	"network-tunneler/internal/network"
	"network-tunneler/pkg/logger"
	"network-tunneler/pkg/netfilter"
)

type NetfilterManager struct {
	manager    *netfilter.Manager
	targetCIDR string
	localPort  string
	logger     logger.Logger
	active     bool
}

type NetfilterParams struct {
	fx.In

	Config *Config
	Logger logger.Logger
}

func NewNetfilterManager(p NetfilterParams) *NetfilterManager {
	return &NetfilterManager{
		manager:    netfilter.NewManager(),
		targetCIDR: p.Config.TargetCIDR,
		localPort:  fmt.Sprintf("%d", p.Config.ListenPort),
		logger:     p.Logger.With(logger.String("component", "netfilter")),
		active:     false,
	}
}

func (nf *NetfilterManager) Setup() error {
	if nf.active {
		nf.logger.Warn("netfilter rules already active")
		return nil
	}

	nf.logger.Info("setting up netfilter rules",
		logger.String("target_cidr", nf.targetCIDR),
		logger.String("local_port", nf.localPort),
	)

	rule := network.RedirectTCPToLocalPort(nf.targetCIDR, nf.localPort)

	if err := nf.manager.AddRule(rule); err != nil {
		return fmt.Errorf("failed to add redirect rule: %w", err)
	}

	if err := nf.manager.Apply(); err != nil {
		return fmt.Errorf("failed to apply netfilter rules: %w", err)
	}

	nf.active = true
	nf.logger.Info("netfilter rules applied successfully")

	return nil
}

func (nf *NetfilterManager) Cleanup() error {
	if !nf.active {
		nf.logger.Debug("netfilter rules not active, nothing to clean up")
		return nil
	}

	nf.logger.Info("cleaning up netfilter rules")

	if err := nf.manager.Remove(); err != nil {
		return fmt.Errorf("failed to remove netfilter rules: %w", err)
	}

	nf.active = false
	nf.logger.Info("netfilter rules removed successfully")

	return nil
}

func (nf *NetfilterManager) IsActive() bool {
	return nf.active
}
