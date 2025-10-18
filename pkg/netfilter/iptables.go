//go:build linux

package netfilter

import (
	"fmt"
	"os/exec"
	"strings"
)

type Manager struct {
	rules []*Rule
}

func NewManager() *Manager {
	return &Manager{
		rules: make([]*Rule, 0),
	}
}

func (m *Manager) AddRule(rule *Rule) error {
	if err := rule.Validate(); err != nil {
		return fmt.Errorf("invalid rule: %w", err)
	}

	m.rules = append(m.rules, rule)
	return nil
}

func (m *Manager) Apply() error {
	for _, rule := range m.rules {
		exists, err := m.CheckRule(rule)
		if err != nil {
			return fmt.Errorf("failed to check rule %s: %w", rule.String(), err)
		}

		if exists {
			continue
		}

		if err := m.insertRule(rule); err != nil {
			return fmt.Errorf("failed to apply rule %s: %w", rule.String(), err)
		}
	}
	return nil
}

func (m *Manager) Remove() error {
	for i := len(m.rules) - 1; i >= 0; i-- {
		rule := m.rules[i]
		if err := m.deleteRule(rule); err != nil {
			return fmt.Errorf("failed to remove rule %s: %w", rule.String(), err)
		}
	}
	return nil
}

func (m *Manager) insertRule(rule *Rule) error {
	args := []string{"-t", string(rule.Table), "-I", string(rule.Chain)}
	args = append(args, rule.Args()[2:]...) // Skip table flag

	cmd := exec.Command("iptables", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("iptables insert failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (m *Manager) deleteRule(rule *Rule) error {
	args := []string{"-t", string(rule.Table), "-D", string(rule.Chain)}
	args = append(args, rule.Args()[2:]...)

	cmd := exec.Command("iptables", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(output), "No chain/target/match by that name") ||
			strings.Contains(string(output), "does a matching rule exist") {
			return nil
		}
		return fmt.Errorf("iptables delete failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (m *Manager) CheckRule(rule *Rule) (bool, error) {
	args := []string{"-t", string(rule.Table), "-C", string(rule.Chain)}
	args = append(args, rule.Args()[2:]...)

	cmd := exec.Command("iptables", args...)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func ListRules(table Table, chain Chain) (string, error) {
	cmd := exec.Command("iptables", "-t", string(table), "-L", string(chain), "-n", "-v", "--line-numbers")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("iptables list failed: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

func FlushChain(table Table, chain Chain) error {
	cmd := exec.Command("iptables", "-t", string(table), "-F", string(chain))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("iptables flush failed: %w, output: %s", err, string(output))
	}

	return nil
}
