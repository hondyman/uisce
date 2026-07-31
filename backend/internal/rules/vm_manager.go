package rules

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// VMManager orchestrates compilation, caching, and execution of rules.
// It is thread-safe and designed to be instantiated once at application
// startup. The cache is keyed by a caller-provided ruleID (typically
// the rule's database primary key).
//
// VMManager depends on both vm primitives (for execution) and rules AST
// types (for compilation), so it lives in the rules package — putting it
// in vm would re-introduce the import cycle the architecture avoids.
type VMManager struct {
	syms  *vm.SymbolDict
	enums *vm.EnumDict
	cache sync.Map // map[string]*vm.CompileResult
	vm    *vm.VM

	// Metrics
	cacheHits     atomic.Uint64
	cacheMisses   atomic.Uint64
	fallbacks     atomic.Uint64
	vmPathCount   atomic.Uint64
	compileErrors atomic.Uint64
}

// NewVMManager initializes a new Manager with empty dictionaries.
func NewVMManager() *VMManager {
	return &VMManager{
		syms:  vm.NewSymbolDict(),
		enums: vm.NewEnumDict(),
		vm:    vm.NewVM(),
	}
}

// NewVMManagerWithDicts initializes a Manager with caller-provided
// dictionaries. Useful when the schema dictionaries are shared across
// multiple evaluators or wired externally.
func NewVMManagerWithDicts(syms *vm.SymbolDict, enums *vm.EnumDict) *VMManager {
	return &VMManager{
		syms:  syms,
		enums: enums,
		vm:    vm.NewVM(),
	}
}

// RegisterAndFreeze pre-populates the symbol and enum dictionaries from
// a rule corpus. This MUST be called at startup before any Evaluate calls.
// Rules containing fields or enums not present in the dictionary will
// fail to compile and fall back to the recursive evaluator.
func (m *VMManager) RegisterAndFreeze(corpus []*RuleNode) {
	for _, rule := range corpus {
		extractPathsAndEnums(rule, m.syms, m.enums)
	}
	m.syms.Freeze()
	m.enums.Freeze()
}

// extractPathsAndEnums walks the AST and Intern's every field path and
// string literal into the dictionaries. Exposed (lowercase) for use
// during incremental schema growth.
func extractPathsAndEnums(node *RuleNode, syms *vm.SymbolDict, enums *vm.EnumDict) {
	if node == nil {
		return
	}
	if node.Type == NodeTypeGroup && node.Group != nil {
		for i := range node.Group.Conditions {
			extractPathsAndEnums(&node.Group.Conditions[i], syms, enums)
		}
		return
	}
	if node.Type == NodeTypeCondition && node.Condition != nil {
		cond := node.Condition
		path := cond.FieldPath
		if path == "" {
			path = cond.Field
		}
		if path != "" {
			_, _ = syms.Intern(path)
		}

		// Intern string values as potential enums.
		if strVal, ok := cond.Value.(string); ok {
			_, _ = enums.Intern(strVal)
		}
		// Intern strings inside 'in' arrays.
		if arr, ok := cond.Value.([]any); ok {
			for _, v := range arr {
				if strVal, ok := v.(string); ok {
					_, _ = enums.Intern(strVal)
				}
			}
		}
	}
}

// EvaluateWithFallback attempts to evaluate the rule using the VM. If the
// rule contains unsupported operators or failed to compile, it invokes
// the provided fallback evaluator.
func (m *VMManager) EvaluateWithFallback(
	ruleID string,
	node *RuleNode,
	input map[string]any,
	fallback func(node *RuleNode, input map[string]any) (bool, error),
) (bool, error) {

	// 1. Check cache for compiled program (sticky unsupported errors).
	var res *vm.CompileResult
	if cached, ok := m.cache.Load(ruleID); ok {
		res = cached.(*vm.CompileResult)
		m.cacheHits.Add(1)
	} else {
		m.cacheMisses.Add(1)
		newRes := CompileVM(node, m.syms, m.enums)
		res = &newRes
		m.cache.Store(ruleID, res)
	}

	// 2. Fallback to recursive evaluator if compile failed.
	if res.Unsupported != nil {
		m.fallbacks.Add(1)
		m.compileErrors.Add(1)
		return fallback(node, input)
	}

	m.vmPathCount.Add(1)

	// 3. Project map -> FastRecord (allocates one record from pool).
	rec := vm.Project(input, m.syms, m.enums)
	defer vm.PutFastRecord(rec)

	// 4. Acquire stack from pool and execute.
	stack := vm.GetStack()
	defer vm.PutStack(stack)

	return m.vm.Run(res.Program, rec, stack), nil
}

// Snapshot returns a point-in-time view of the manager's counters.
type VMSnapshot struct {
	CacheHits     uint64
	CacheMisses   uint64
	Fallbacks     uint64
	VMPathCount   uint64
	CompileErrors uint64
	CacheSize     int
}

func (m *VMManager) Snapshot() VMSnapshot {
	cacheSize := 0
	m.cache.Range(func(_, _ any) bool {
		cacheSize++
		return true
	})
	return VMSnapshot{
		CacheHits:     m.cacheHits.Load(),
		CacheMisses:   m.cacheMisses.Load(),
		Fallbacks:     m.fallbacks.Load(),
		VMPathCount:   m.vmPathCount.Load(),
		CompileErrors: m.compileErrors.Load(),
		CacheSize:     cacheSize,
	}
}

// Symbols returns the underlying SymbolDict (for external callers that
// need to perform validations or pre-intern additional paths).
func (m *VMManager) Symbols() *vm.SymbolDict { return m.syms }

// Enums returns the underlying EnumDict.
func (m *VMManager) Enums() *vm.EnumDict { return m.enums }

// EvaluateJSONWithFallback is the recommended hot path for ingestion:
// raw JSON bytes are decoded directly into a FastRecord via the streaming
// decoder, bypassing map[string]any entirely. On compile-failure fallback,
// the JSON is unmarshalled into a map (rare path; alloc-heavy but
// acceptable since it only triggers for unsupported operators).
func (m *VMManager) EvaluateJSONWithFallback(
	ruleID string,
	node *RuleNode,
	jsonData []byte,
	fallback func(node *RuleNode, input map[string]any) (bool, error),
) (bool, error) {

	// 1. Check cache for compiled program.
	var res *vm.CompileResult
	if cached, ok := m.cache.Load(ruleID); ok {
		res = cached.(*vm.CompileResult)
		m.cacheHits.Add(1)
	} else {
		m.cacheMisses.Add(1)
		newRes := CompileVM(node, m.syms, m.enums)
		res = &newRes
		m.cache.Store(ruleID, res)
	}

	// 2. Fallback to recursive evaluator if compile failed.
	if res.Unsupported != nil {
		m.fallbacks.Add(1)
		m.compileErrors.Add(1)

		// Fallback requires map[string]any. Decode JSON here (rare path).
		var input map[string]any
		if err := json.Unmarshal(jsonData, &input); err != nil {
			return false, err
		}
		return fallback(node, input)
	}

	m.vmPathCount.Add(1)

	// 3. Decode JSON directly to FastRecord (~0 allocations via pool).
	rec, err := vm.DecodeJSON(jsonData, m.syms, m.enums)
	if err != nil {
		return false, err
	}
	defer vm.PutFastRecord(rec)

	// 4. Acquire stack from pool and execute.
	stack := vm.GetStack()
	defer vm.PutStack(stack)

	return m.vm.Run(res.Program, rec, stack), nil
}