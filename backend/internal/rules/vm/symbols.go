package vm

import (
	"errors"
	"sync"
	"sync/atomic"
)

// SymbolDict maps dotted field paths ("customer.account.tier") to
// densely-packed uint32 IDs used as array indices into FastRecord.
type SymbolDict struct {
	mu     sync.RWMutex
	index  map[string]uint32
	paths  []string
	frozen atomic.Bool
}

func NewSymbolDict() *SymbolDict {
	return &SymbolDict{
		index: make(map[string]uint32),
		paths: make([]string, 0, 64),
	}
}

// Intern returns the existing ID for path or assigns a new one.
// Safe to call concurrently before Freeze(); returns ErrFrozen after.
func (d *SymbolDict) Intern(path string) (uint32, error) {
	if d.frozen.Load() {
		return 0, ErrFrozen
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if id, ok := d.index[path]; ok {
		return id, nil
	}
	id := uint32(len(d.paths))
	d.index[path] = id
	d.paths = append(d.paths, path)
	return id, nil
}

// Resolve returns the ID for a previously-interned path.
// Hot path — must be O(1) with no allocation.
func (d *SymbolDict) Resolve(path string) (uint32, bool) {
	if d.frozen.Load() {
		id, ok := d.index[path]
		return id, ok
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	id, ok := d.index[path]
	return id, ok
}

// ResolveBytes is the zero-allocation variant of Resolve for callers that
// hold the path as a []byte (e.g., the JSON streaming decoder). The map
// lookup internally converts []byte to string; the compiler elides this
// allocation when the key is not stored.
func (d *SymbolDict) ResolveBytes(path []byte) (uint32, bool) {
	if d.frozen.Load() {
		id, ok := d.index[string(path)]
		return id, ok
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	id, ok := d.index[string(path)]
	return id, ok
}

// Freeze seals the dictionary. After Freeze(), Resolve() uses a
// lock-free fast path.
func (d *SymbolDict) Freeze() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.frozen.Store(true)
}

// Num returns the highest assigned ID + 1 (== FastRecord capacity).
func (d *SymbolDict) Num() uint32 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return uint32(len(d.paths))
}

// Path returns the string path for a given ID (debug/trace use only).
func (d *SymbolDict) Path(id uint32) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if int(id) < len(d.paths) {
		return d.paths[id]
	}
	return ""
}

var ErrFrozen = errors.New("symbol dict is frozen")