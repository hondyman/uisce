package bp

// CompileBlueprint transforms a list of BPSteps into a navigable WorkflowBlueprint
func CompileBlueprint(steps []BPStep) *WorkflowBlueprint {
	nodeMap := make(map[string]*CompiledNode)

	// Pre-map for quick lookups by ID/Key if needed, though steps usually have ID.
	// We'll use StepKey as the 'ID' for mapping if explicit IDs aren't reliable in the input,
	// but the SQL return has IDs. Let's use internal ID for the map, but NextNodes will also need to be IDs.
	// Since the DB uses IDs for foreign keys, we assume we have them.
	// HOWEVER, the `BPStep` struct from `designer.go` input might NOT have IDs if it's the `SaveInput`,
	// but `Execute` will always load from DB, so we have IDs.
	//
	// Implicit Sequencing:
	// A simple approach for the provided schema (which uses `seq` and `step_key`)
	// is to link Step N to N+1 by default unless it's a branching node.
	// The User Request example implies a graph with edges, but our DB schema `business_process_step`
	// doesn't have an explicit 'edges' table, it relies on `seq` OR `routing_rules`.
	//
	// FOR NOW: We will link Step[i] to Step[i+1] as the default 'NextNode',
	// mimicking a sequence. Branching nodes will override this behavior dynamically
	// in the workflow executor, BUT the compiler needs to provide the mapping.
	//
	// Wait, the User Request explicitly says: "Convert nodes + edges to a compiled blueprint".
	// But our `business_process_step` table (from `designer.go`) does NOT have an edges column,
	// nor is there an `edges` table.
	//
	// `SaveDesigner` takes `Steps []BPStepPayload` and saves them.
	// `GetDesigner` shows steps.
	//
	// If the frontend sends "Nodes" and "Edges" to `SaveDesigner`, we currently discard edges
	// and flatten to a list.
	//
	// FIX: The backend persistence implementation in `designer.go` assumes a FLAT SEQUENCE (`seq`).
	// To support branching, we rely on `RoutingRules` inside a step to point to other steps (by key or ID).
	//
	// So, `CompileBlueprint` will:
	// 1. Map all steps by ID (and StepKey for reference).
	// 2. Determine `NextNodes`:
	//    - Default: The next step in the `steps` slice (ordered by Seq).
	//    - Branch/Routing: The executor will look up the target by Key/ID from the `RoutingRules`.
	//      The `NextNodes` field might be redundant if routing is purely dynamic,
	//      but we can populate it with the default "next" for visualization or default path.

	idToIndex := make(map[string]int)
	for i, s := range steps {
		idToIndex[s.ID] = i
	}

	for i, s := range steps {
		var nextNodes []string

		// Logic: If there is a next step in sequence, add it.
		// Real branching logic (e.g. "go to step X") happens at runtime via RoutingRules,
		// relying on `workflow_executor.go` to handle `ResolveBranchActivity`.
		if i < len(steps)-1 {
			nextNodes = append(nextNodes, steps[i+1].ID)
		}

		nodeMap[s.ID] = &CompiledNode{
			StepKey:       s.StepKey,
			Type:          s.Type,
			ActivityName:  s.ActivityName,
			SignalName:    s.SignalName,
			ConditionExpr: s.ConditionExpr,
			DelayExpr:     s.DelayExpr,
			SLAExpr:       s.SLAExpr,
			ApprovalChain: s.ApprovalChain,
			RoutingRules:  s.RoutingRules,
			Escalations:   s.Escalations,
			NextNodes:     nextNodes,
		}
	}

	startID := ""
	endID := ""
	if len(steps) > 0 {
		startID = steps[0].ID
		endID = steps[len(steps)-1].ID
	}

	return &WorkflowBlueprint{
		Nodes:   nodeMap,
		StartID: startID,
		EndID:   endID,
	}
}
