package netfilter

type RuleBuilder struct {
	rule *Rule
}

func NewRule() *RuleBuilder {
	return &RuleBuilder{
		rule: &Rule{},
	}
}

func (rb *RuleBuilder) Table(t Table) *RuleBuilder {
	rb.rule.Table = t
	return rb
}

func (rb *RuleBuilder) Chain(c Chain) *RuleBuilder {
	rb.rule.Chain = c
	return rb
}

func (rb *RuleBuilder) Protocol(p Protocol) *RuleBuilder {
	rb.rule.Protocol = p
	return rb
}

func (rb *RuleBuilder) Source(cidr string) *RuleBuilder {
	rb.rule.Source = cidr
	return rb
}

func (rb *RuleBuilder) Destination(cidr string) *RuleBuilder {
	rb.rule.Destination = cidr
	return rb
}

func (rb *RuleBuilder) SrcPort(port string) *RuleBuilder {
	rb.rule.SrcPort = port
	return rb
}

func (rb *RuleBuilder) DstPort(port string) *RuleBuilder {
	rb.rule.DstPort = port
	return rb
}

func (rb *RuleBuilder) Target(t Target) *RuleBuilder {
	rb.rule.Target = t
	return rb
}

func (rb *RuleBuilder) ToPort(port string) *RuleBuilder {
	rb.rule.ToPort = port
	return rb
}

func (rb *RuleBuilder) Comment(comment string) *RuleBuilder {
	rb.rule.Comment = comment
	return rb
}

func (rb *RuleBuilder) Build() (*Rule, error) {
	if err := rb.rule.Validate(); err != nil {
		return nil, err
	}
	return rb.rule, nil
}

func (rb *RuleBuilder) MustBuild() *Rule {
	rule, err := rb.Build()
	if err != nil {
		panic(err)
	}
	return rule
}
