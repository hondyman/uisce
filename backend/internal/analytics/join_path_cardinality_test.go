package analytics

import "testing"

// These cover TraversalCardinality (down-cardinality: does a root row
// fan out to many target rows) and RootOwnership (up-cardinality: does a
// target row trace back to more than one root row) against the path
// shapes from the fan-out cardinality trace. Neither function had test
// coverage before this file - both were exercised only implicitly via
// buildMultiBOSQL's behavior.
func step(cardinality string) JoinPathStep {
	return JoinPathStep{Cardinality: cardinality}
}

// Shape A: root -(M:1)-> a -(M:1)-> b. No to-many hop anywhere (down), so
// TraversalCardinality never flags this as expanding - no aggregation
// logic in buildMultiBOSQL would even trigger. But both hops are M:1, so
// RootOwnership must be "shared": b is reachable from every root whose
// "a" (whatever that root's own a is) eventually reaches the same b -
// e.g. two orders for different customers in the same region both reach
// that region's row.
func TestCardinality_ShapeA_AllToOneChain(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("M:1"), step("M:1")}}
	if got := path.TraversalCardinality(); got != "one" {
		t.Errorf("TraversalCardinality: want %q, got %q", "one", got)
	}
	if got := path.RootOwnership(); got != "shared" {
		t.Errorf("RootOwnership: want %q, got %q", "shared", got)
	}
}

// Shape B: root -(1:M)-> line_items. Single to-many hop, no to-many
// hop in the up direction (a "1:M" step's up-cardinality is 1: each
// line_item belongs to exactly one root).
func TestCardinality_ShapeB_SingleHopToMany(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("1:M")}}
	if got := path.TraversalCardinality(); got != "many" {
		t.Errorf("TraversalCardinality: want %q, got %q", "many", got)
	}
	if got := path.RootOwnership(); got != "unique" {
		t.Errorf("RootOwnership: want %q, got %q", "unique", got)
	}
}

// Shape C: root -(M:1)-> a -(1:M)-> b. Composed down-cardinality is
// "many" (the second hop fans out), and RootOwnership is "shared" via
// the first hop alone: many roots can map to the same "a", and every
// root sharing that "a" reaches the identical set of b rows.
func TestCardinality_ShapeC_MixedHop_NToOneThenOneToMany(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("M:1"), step("1:M")}}
	if got := path.TraversalCardinality(); got != "many" {
		t.Errorf("TraversalCardinality: want %q, got %q", "many", got)
	}
	if got := path.RootOwnership(); got != "shared" {
		t.Errorf("RootOwnership: want %q, got %q", "shared", got)
	}
}

// Shape D: root -(1:M)-> line_items -(1:M)-> allocations. Every hop's
// up-cardinality is 1, so despite two hops of fan-out, ownership stays
// unique: every allocation determines exactly one line_item, which
// determines exactly one root.
func TestCardinality_ShapeD_ChainOfToMany(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("1:M"), step("1:M")}}
	if got := path.TraversalCardinality(); got != "many" {
		t.Errorf("TraversalCardinality: want %q, got %q", "many", got)
	}
	if got := path.RootOwnership(); got != "unique" {
		t.Errorf("RootOwnership: want %q, got %q", "unique", got)
	}
}

// M:M is shared regardless of position - it fans out in both directions
// at once.
func TestCardinality_ManyToMany_IsSharedAndMany(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("M:M")}}
	if got := path.TraversalCardinality(); got != "many" {
		t.Errorf("TraversalCardinality: want %q, got %q", "many", got)
	}
	if got := path.RootOwnership(); got != "shared" {
		t.Errorf("RootOwnership: want %q, got %q", "shared", got)
	}
}

// A pure 1:1 chain: no fan-out in either direction.
func TestCardinality_OneToOneChain_UniqueAndOne(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("1:1"), step("1:1")}}
	if got := path.TraversalCardinality(); got != "one" {
		t.Errorf("TraversalCardinality: want %q, got %q", "one", got)
	}
	if got := path.RootOwnership(); got != "unique" {
		t.Errorf("RootOwnership: want %q, got %q", "unique", got)
	}
}

// TestCardinality_UnrecognizedHop_IsUnresolvedNotUnique is the negative
// test for the collapse this function used to be vulnerable to: a step
// whose Cardinality the resolver couldn't classify (empty, or some
// string outside the known four) must come back "unresolved", never
// silently "unique". A prior version of this function fell through an
// unrecognized cardinality straight to "unique" - this is the test that
// would have caught it, since TestCardinality_ShapeB/D above only ever
// exercise recognized values and would have passed regardless.
func TestCardinality_UnrecognizedHop_IsUnresolvedNotUnique(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("1:M"), step("")}}
	if got := path.TraversalCardinality(); got != "many" {
		t.Errorf("TraversalCardinality: want %q, got %q", "many", got)
	}
	if got := path.RootOwnership(); got != "unresolved" {
		t.Fatalf("RootOwnership: want %q (unrecognized hop must not collapse to \"unique\"), got %q", "unresolved", got)
	}
}

// An unresolved hop must not mask a DEFINITE "shared" finding elsewhere
// in the same path - "shared" is already at least as bad as anything
// "unresolved" could turn out to be, so it must win.
func TestCardinality_UnresolvedHop_DoesNotMaskDefiniteShared(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step(""), step("M:1")}}
	if got := path.RootOwnership(); got != "shared" {
		t.Fatalf("RootOwnership: want %q (a definite M:1 hop must win over an unresolved one), got %q", "shared", got)
	}
}

// TestCardinality_UnrecognizedHop_TraversalIsUnresolvedNotOne is
// TraversalCardinality's own version of the negative test above:
// TraversalCardinality used to fall through an unrecognized hop straight
// to "one" (the exact same collapse RootOwnership had, one function
// over) - a query whose row grain the resolver couldn't classify would
// report "no fan-out" rather than "don't know," and a consumer branching
// on that would apply the no-fan-out remedy to a possibly-fanning path.
func TestCardinality_UnrecognizedHop_TraversalIsUnresolvedNotOne(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("1:1"), step("")}}
	if got := path.TraversalCardinality(); got != "unresolved" {
		t.Fatalf("TraversalCardinality: want %q (unrecognized hop must not collapse to \"one\"), got %q", "unresolved", got)
	}
}

// A definite "many" (down) hop still wins over an unresolved one
// elsewhere in the path, mirroring TestCardinality_UnresolvedHop_DoesNotMaskDefiniteShared.
func TestCardinality_UnresolvedHop_DoesNotMaskDefiniteMany(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step(""), step("1:M")}}
	if got := path.TraversalCardinality(); got != "many" {
		t.Fatalf("TraversalCardinality: want %q (a definite 1:M hop must win over an unresolved one), got %q", "many", got)
	}
}

// TestAnalyze_BothAxesIndependent pins the shape from the fan-out
// cardinality trace's own worked example: an "M:1" hop is "one" under
// TraversalCardinality (no down fan-out) and "shared" under RootOwnership
// (up fan-out) AT THE SAME TIME, out of the SAME single fold - the two
// facts are independent, not two spellings of one verdict.
func TestAnalyze_BothAxesIndependent(t *testing.T) {
	path := &JoinPath{Steps: []JoinPathStep{step("M:1")}}
	got := path.Analyze()
	if got.Cardinality != "one" || got.Ownership != "shared" {
		t.Fatalf("Analyze(): want {Cardinality: one, Ownership: shared}, got %+v", got)
	}
	// And the two convenience methods must agree with the fold exactly -
	// they're wrappers over it now, not a second implementation.
	if path.TraversalCardinality() != got.Cardinality || path.RootOwnership() != got.Ownership {
		t.Fatalf("TraversalCardinality()/RootOwnership() disagree with Analyze(): %q/%q vs %+v",
			path.TraversalCardinality(), path.RootOwnership(), got)
	}
}

// A nil path (e.g. same driving table as root - see buildMultiBOSQL's
// "empty path when from==to" comment) is trivially one root, one row:
// "one" and "unique".
func TestCardinality_NilPath_OneAndUnique(t *testing.T) {
	var path *JoinPath
	if got := path.TraversalCardinality(); got != "one" {
		t.Errorf("TraversalCardinality: want %q, got %q", "one", got)
	}
	if got := path.RootOwnership(); got != "unique" {
		t.Errorf("RootOwnership: want %q, got %q", "unique", got)
	}
}
