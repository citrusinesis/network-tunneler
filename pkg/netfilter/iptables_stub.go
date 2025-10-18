//go:build !linux

package netfilter

import "fmt"

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
	return fmt.Errorf("netfilter is only supported on Linux")
}

func (m *Manager) Remove() error {
	return fmt.Errorf("netfilter is only supported on Linux")
}

func (m *Manager) CheckRule(rule *Rule) (bool, error) {
	return false, fmt.Errorf("netfilter is only supported on Linux")
}

func ListRules(table Table, chain Chain) (string, error) {
	return "", fmt.Errorf("netfilter is only supported on Linux")
}

func FlushChain(table Table, chain Chain) error {
	return fmt.Errorf("netfilter is only supported on Linux")
}
