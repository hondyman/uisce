package datapipeline

import (
	"context"
	"database/sql"
	"fmt"
)

// Deps are the environment's connections. A nil dependency makes the nodes
// that need it fail with a plain message at run start, never mid-run.
type Deps struct {
	Rules     RuleChecker // rule_check
	BO        BOClient    // bo_source, bo_sink
	Files     FileEngine  // file_source, file_sink
	StagingDB *sql.DB     // staging_sink / master_sink data plane
	// AlphaDB is the control-plane pool (attribute_def, semantic survivorship).
	// Required for master_sink.
	AlphaDB *sql.DB
	// Queues powers queue_source / queue_sink (Kafka/Redpanda/SQS/Service Bus).
	Queues QueueBroker
	// Bindings resolves staging table bindings for rule checks in front of
	// a staging load (nil: such checks can't run).
	Bindings BindingSource
}

// BindingSource resolves the approved binding of a staging table to a
// business object: field -> staging column, nil when there is none.
type BindingSource interface {
	StagingFields(ctx context.Context, tenantID, boKey, table string) (map[string]string, error)
}

// Factory builds each node's runtime from Deps.
func (d Deps) Source(n Node) (Source, error) {
	switch n.Type {
	case NodeFileSource:
		return newFileSource(n, d.Files)
	case NodeBOSource:
		return newBOSource(n, d.BO)
	case NodeQueueSource:
		return newQueueSource(n, d.Queues)
	}
	return nil, fmt.Errorf("%q is not a source", n.Type)
}

func (d Deps) Processor(n Node) (Processor, error) {
	switch n.Type {
	case NodeValidate:
		return newValidateProc(n)
	case NodeMap:
		return newMapProc(n)
	case NodeRuleCheck:
		return newRuleCheckProc(n, d.Rules, d.Bindings)
	case NodeBOSink:
		return newBOSinkProc(n, d.BO)
	case NodeStagingSink:
		return newStagingSink(n, d.StagingDB)
	case NodeMasterSink:
		return newMasterSink(n, d)
	case NodeQueueSink:
		return newQueueSink(n, d.Queues)
	case NodeFileSink:
		return newFileSink(n, d.Files)
	}
	return nil, fmt.Errorf("%q is not a processing step", n.Type)
}
