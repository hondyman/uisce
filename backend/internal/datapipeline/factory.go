package datapipeline

import (
	"database/sql"
	"fmt"
)

// Deps are the environment's connections. A nil dependency makes the nodes
// that need it fail with a plain message at run start, never mid-run.
type Deps struct {
	Rules     RuleChecker // rule_check
	BO        BOClient    // bo_source, bo_sink
	Files     FileEngine  // file_source, file_sink
	StagingDB *sql.DB     // staging_sink
}

// Factory builds each node's runtime from Deps.
func (d Deps) Source(n Node) (Source, error) {
	switch n.Type {
	case NodeFileSource:
		return newFileSource(n, d.Files)
	case NodeBOSource:
		return newBOSource(n, d.BO)
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
		return newRuleCheckProc(n, d.Rules)
	case NodeBOSink:
		return newBOSinkProc(n, d.BO)
	case NodeStagingSink:
		return newStagingSink(n, d.StagingDB)
	case NodeFileSink:
		return newFileSink(n, d.Files)
	}
	return nil, fmt.Errorf("%q is not a processing step", n.Type)
}
