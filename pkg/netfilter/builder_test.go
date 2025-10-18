package netfilter_test

import (
	"strings"
	"testing"

	"network-tunneler/pkg/netfilter"
)

func TestRuleBuilder(t *testing.T) {
	rule, err := netfilter.NewRule().
		Table(netfilter.TableNat).
		Chain(netfilter.ChainOutput).
		Protocol(netfilter.ProtocolTCP).
		Destination("100.64.0.0/10").
		Target(netfilter.TargetRedirect).
		ToPort("9999").
		Comment("my-custom-rule").
		Build()

	if err != nil {
		t.Fatalf("failed to build rule: %v", err)
	}

	expected := "table=nat chain=OUTPUT proto=tcp dst=100.64.0.0/10 target=REDIRECT toport=9999 comment=\"my-custom-rule\""
	if got := rule.String(); got != expected {
		t.Errorf("rule.String() = %q, want %q", got, expected)
	}
}

func TestNewRule(t *testing.T) {
	rule := netfilter.NewRule().
		Table(netfilter.TableNat).
		Chain(netfilter.ChainOutput).
		Destination("192.168.1.0/24").
		Target(netfilter.TargetRedirect).
		ToPort("8080").
		MustBuild()

	expected := "table=nat chain=OUTPUT dst=192.168.1.0/24 target=REDIRECT toport=8080"
	if got := rule.String(); got != expected {
		t.Errorf("rule.String() = %q, want %q", got, expected)
	}
}

func TestRuleMustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustBuild to panic on invalid rule, but it didn't")
		}
	}()

	_ = netfilter.NewRule().MustBuild()
}

func TestRuleBuilder_InvalidDestinationCIDR(t *testing.T) {
	_, err := netfilter.NewRule().
		Table(netfilter.TableNat).
		Chain(netfilter.ChainOutput).
		Destination("invalid-cidr").
		Target(netfilter.TargetRedirect).
		ToPort("9999").
		Build()

	if err == nil {
		t.Fatal("expected error for invalid destination CIDR, got nil")
	}

	expectedErr := "invalid destination CIDR"
	if !containsString(err.Error(), expectedErr) {
		t.Errorf("error message = %q, want to contain %q", err.Error(), expectedErr)
	}
}

func TestRuleBuilder_InvalidSourceCIDR(t *testing.T) {
	_, err := netfilter.NewRule().
		Table(netfilter.TableNat).
		Chain(netfilter.ChainOutput).
		Source("not-an-ip").
		Destination("192.168.1.0/24").
		Target(netfilter.TargetAccept).
		Build()

	if err == nil {
		t.Fatal("expected error for invalid source CIDR, got nil")
	}

	expectedErr := "invalid source CIDR"
	if !containsString(err.Error(), expectedErr) {
		t.Errorf("error message = %q, want to contain %q", err.Error(), expectedErr)
	}
}

func TestRuleBuilder_ValidCIDRFormats(t *testing.T) {
	tests := []struct {
		name string
		cidr string
	}{
		{"full CIDR", "100.64.0.0/10"},
		{"subnet", "192.168.1.0/24"},
		{"single IP", "192.168.1.1"},
		{"host /32", "10.0.0.1/32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := netfilter.NewRule().
				Table(netfilter.TableNat).
				Chain(netfilter.ChainOutput).
				Destination(tt.cidr).
				Target(netfilter.TargetRedirect).
				ToPort("9999").
				Build()

			if err != nil {
				t.Errorf("unexpected error for valid CIDR %q: %v", tt.cidr, err)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
