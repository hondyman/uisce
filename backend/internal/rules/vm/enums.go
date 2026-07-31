package vm

import (
	"errors"
	"sync"
	"sync/atomic"
)

// EnumDict interns categorical string literals ("GOLD", "ACTIVE") into
// uint32 IDs so equality becomes an integer compare instead of a string compare.
type EnumDict struct {
	mu     sync.RWMutex
	values map[string]uint32
	frozen atomic.Bool
}

func NewEnumDict() *EnumDict {
	return &EnumDict{
		values: make(map[string]uint32),
	}
}

// Intern returns the existing ID for lit or assigns a new one.
// Safe to call concurrently before Freeze(); returns ErrEnumFrozen after.
func (d *EnumDict) Intern(lit string) (uint32, error) {
	if d.frozen.Load() {
		return 0, ErrEnumFrozen
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if id, ok := d.values[lit]; ok {
		return id, nil
	}
	// Start enum IDs at 1 to reserve 0 for EnumMissing
	id := uint32(len(d.values) + 1)
	d.values[lit] = id
	return id, nil
}

// ID returns the integer ID for a previously interned literal.
// Hot path — lock-free after Freeze().
func (d *EnumDict) ID(lit string) (uint32, bool) {
	if d.frozen.Load() {
		id, ok := d.values[lit]
		return id, ok
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	id, ok := d.values[lit]
	return id, ok
}

// Freeze seals the dictionary.
func (d *EnumDict) Freeze() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.frozen.Store(true)
}

var ErrEnumFrozen = errors.New("enum dict is frozen")