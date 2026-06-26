package semlayer.governance.pipelines

import future.keywords.if
import future.keywords.in

default allow = false

# Allow if no deny reasons
allow if count(deny) == 0

# Deny if the pipeline has no steps
deny contains msg if {
	input.nodes
	count(input.nodes) == 0
	msg := "Pipeline must have at least one node"
}

# Deny if there is no Start node
deny contains msg if {
	input.nodes
	not has_start_node(input.nodes)
	msg := "Pipeline must have a Start node"
}

has_start_node(nodes) if {
	some node in nodes
	node.type == "start" # Adjust based on actual node type for start
}
