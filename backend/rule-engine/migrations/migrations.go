package migrations

import "github.com/hondyman/semlayer/backend/rule-engine/runtime"

type Migration struct {
	Version string
	Apply   func(runtime.RuleNode) runtime.RuleNode
}

var Migrations = []Migration{
	{
		Version: "1.1.0",
		Apply: func(rule runtime.RuleNode) runtime.RuleNode {
			if rule.Type == "Condition" && rule.Condition.Operator == "eq" {
				rule.Condition.Operator = "equals"
			}
			return rule
		},
	},
}

func Migrate(rule runtime.RuleNode) runtime.RuleNode {
	for _, m := range Migrations {
		rule = m.Apply(rule)
	}
	return rule
}
