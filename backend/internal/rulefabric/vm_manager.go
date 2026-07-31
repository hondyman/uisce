package rulefabric

import (
	"sync"
	"sync/atomic"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// VMManager orchestrates compilation, caching, and execution of rulefabric
// rules. It is thread-safe and designed to be instantiated once at startup.
//
// The cache key is the rule's UUID string. Compile failures (unsupported
// operators, missing field paths, etc.) are sticky: a rule that fails
// once is never retried, and every Evaluate call falls back to the
// recursive evaluator.
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

// NewVMManagerWithDicts initializes a Manager with caller-provided dicts.
func NewVMManagerWithDicts(syms *vm.SymbolDict, enums *vm.EnumDict) *VMManager {
	return &VMManager{
		syms:  syms,
		enums: enums,
		vm:    vm.NewVM(),
	}
}

// RegisterAndFreeze pre-populates the symbol and enum dictionaries from a
// rule corpus. Must be called at startup before any Evaluate calls.
func (m *VMManager) RegisterAndFreeze(corpus []*RuleWithLogic) {
	for _, rule := range corpus {
		if len(rule.Logic.ConditionJSON) > 0 {
			var group ConditionGroup
			if err := jsonUnmarshal(rule.Logic.ConditionJSON, &group); err == nil {
				extractRuleFabricPaths(&group, m.syms, m.enums)
			}
		}
	}
	m.syms.Freeze()
	m.enums.Freeze()
}

// ExtractRuleFabricPaths walks a condition tree and Intern's every field
// path and string literal into the dictionaries. Exposed for incremental
// schema growth (e.g., adding a rule at runtime).
func ExtractRuleFabricPaths(g *ConditionGroup, syms *vm.SymbolDict, enums *vm.EnumDict) {
	extractRuleFabricPaths(g, syms, enums)
}

func extractRuleFabricPaths(g *ConditionGroup, syms *vm.SymbolDict, enums *vm.EnumDict) {
	if g == nil {
		return
	}
	for i := range g.Conditions {
		elem := g.Conditions[i]
		m, ok := elem.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := m["type"].(string)
		switch condType {
		case "group":
			var sub ConditionGroup
			if buf, err := jsonMarshal(m); err == nil {
				if err := jsonUnmarshal(buf, &sub); err == nil {
					extractRuleFabricPaths(&sub, syms, enums)
				}
			}
		case "condition":
			var cond Condition
			if buf, err := jsonMarshal(m); err == nil {
				if err := jsonUnmarshal(buf, &cond); err == nil {
					if cond.Field != "" {
						_, _ = syms.Intern(cond.Field)
					}
					if cond.EntityPath != nil && cond.EntityPath.Field != "" {
						_, _ = syms.Intern(cond.EntityPath.Field)
					}
					if s, ok := cond.Value.(string); ok {
						_, _ = enums.Intern(s)
					}
					if arr, ok := cond.Value.([]interface{}); ok {
						for _, v := range arr {
							if s, ok := v.(string); ok {
								_, _ = enums.Intern(s)
							}
						}
					}
				}
			}
		}
	}
}

// EvaluateWithFallback attempts to evaluate the rule via the VM. On any
// failure (compile error, missing field, unsupported operator), it invokes
// the provided fallback evaluator (typically the legacy recursive path).
//
// The `data` map is the rulefabric EvaluationContext.Data; `related` is
// EvaluationContext.RelatedData. Both are flattened into a single
// FastRecord with the `related.` prefix prepended to related-entity
// field paths.
func (m *VMManager) EvaluateWithFallback(
	rule *RuleWithLogic,
	root *ConditionGroup,
	data map[string]any,
	related map[string]any,
	fallback func() (bool, EvaluationDetails),
) (bool, EvaluationDetails) {
	details := EvaluationDetails{
		OperandValues:    make(map[string]any),
		ConditionResults: []ConditionResult{},
	}

	ruleID := rule.ID.String()

	// 1. Cache lookup.
	var res *vm.CompileResult
	if cached, ok := m.cache.Load(ruleID); ok {
		res = cached.(*vm.CompileResult)
		m.cacheHits.Add(1)
	} else {
		m.cacheMisses.Add(1)
		newRes := CompileRuleFabric(rule, root, m.syms, m.enums)
		res = &newRes
		m.cache.Store(ruleID, res)
	}

	// 2. Fallback if compile failed.
	if res.Unsupported != nil {
		m.fallbacks.Add(1)
		m.compileErrors.Add(1)
		return fallback()
	}

	m.vmPathCount.Add(1)

	// 3. Project (data + related) -> FastRecord.
	merged := mergeContextMaps(data, related)
	rec := vm.Project(merged, m.syms, m.enums)
	defer vm.PutFastRecord(rec)

	// 4. Acquire stack and execute.
	stack := vm.GetStack()
	defer vm.PutStack(stack)

	passed := m.vm.Run(res.Program, rec, stack)
	return passed, details
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

// Symbols returns the underlying SymbolDict.
func (m *VMManager) Symbols() *vm.SymbolDict { return m.syms }

// Enums returns the underlying EnumDict.
func (m *VMManager) Enums() *vm.EnumDict { return m.enums }

// mergeContextMaps flattens (data, related) into a single map with
// "related.<field>" prefix on related keys. The VM compiler treats the
// flat map as the source of truth.
func mergeContextMaps(data, related map[string]any) map[string]any {
	if len(related) == 0 {
		return data
	}
	merged := make(map[string]any, len(data)+len(related))
	for k, v := range data {
		merged[k] = v
	}
	for k, v := range related {
		merged["related."+k] = v
	}
	return merged
}