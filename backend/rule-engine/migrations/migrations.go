package migrations

import vm "github.com/hondyman/uisce/backend/internal/rules/vm"

type Migration struct {
	Version string
	Apply   func(vm.RuleNode) vm.RuleNode
}

var Migrations = []Migration{
	{
		Version: "1.1.0",
		Apply: func(rule vm.RuleNode) vm.RuleNode {
			if rule.Type == vm.NodeTypeCondition && rule.Condition.Operator == "eq" {
				rule.Condition.Operator = "equals"
			}
			return rule
		},
	},
}

func Migrate(rule vm.RuleNode) vm.RuleNode {
	for _, m := range Migrations {
		rule = m.Apply(rule)
	}
	return rule
}
