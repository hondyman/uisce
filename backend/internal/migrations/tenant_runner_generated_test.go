package migrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// A generated app (ADR-050) gets the same lock, ledger and drift rule as a directory of files.

func gen(files ...GeneratedFile) func(Target) ([]GeneratedFile, bool, error) {
	return func(t Target) ([]GeneratedFile, bool, error) {
		if t.App != "structure" {
			return nil, false, nil
		}
		return files, true, nil
	}
}

func TestSource_GeneratedFilesAreListedHashedAndReadFromMemory(t *testing.T) {
	r := &TenantRunner{Root: t.TempDir(), Generated: gen(
		GeneratedFile{Filename: "0002_b.up.sql", SQL: "select 2;"},
		GeneratedFile{Filename: "0001_a.up.sql", SQL: "select 1;"},
	)}
	names, shas, read, err := r.source(target("structure"))
	require.NoError(t, err)
	require.Equal(t, []string{"0001_a.up.sql", "0002_b.up.sql"}, names, "sorted: that is the apply order")
	sum := sha256.Sum256([]byte("select 1;"))
	require.Equal(t, hex.EncodeToString(sum[:]), shas["0001_a.up.sql"], "the same sha256 a file with this content would have")
	body, err := read("0002_b.up.sql")
	require.NoError(t, err)
	require.Equal(t, "select 2;", body)
}

func TestSource_AnAppTheGeneratorDoesNotOwnStillComesFromItsDirectory(t *testing.T) {
	r := appDir(t, map[string]string{"0001_x.up.sql": "select 1;"})
	r.Generated = gen(GeneratedFile{Filename: "0001_a.up.sql", SQL: "select 1;"})
	names, _, read, err := r.source(target("orm"))
	require.NoError(t, err)
	require.Equal(t, []string{"0001_x.up.sql"}, names)
	b, err := read("0001_x.up.sql")
	require.NoError(t, err)
	require.Equal(t, "select 1;", b)
}

func TestSource_AGeneratedAppNeverFallsThroughToADirectory(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "structure"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "structure", "0001_disk.up.sql"), []byte("select 1;"), 0o644))
	r := &TenantRunner{Root: root, Generated: gen(GeneratedFile{Filename: "0001_mem.up.sql", SQL: "select 2;"})}
	names, _, _, err := r.source(target("structure"))
	require.NoError(t, err)
	require.Equal(t, []string{"0001_mem.up.sql"}, names, "the generated source answers for its app; a stray directory of the same name is ignored")
}

func TestSource_RefusesBadGeneratedFiles(t *testing.T) {
	for name, files := range map[string][]GeneratedFile{
		"not an up file":   {{Filename: "x.sql", SQL: "select 1;"}},
		"a path":           {{Filename: "../0001.up.sql", SQL: "select 1;"}},
		"a nested path":    {{Filename: "a/0001.up.sql", SQL: "select 1;"}},
		"a backslash path": {{Filename: `a\0001.up.sql`, SQL: "select 1;"}},
		"a duplicate":      {{Filename: "0001.up.sql", SQL: "a"}, {Filename: "0001.up.sql", SQL: "b"}},
	} {
		_, _, _, err := (&TenantRunner{Root: t.TempDir(), Generated: gen(files...)}).source(target("structure"))
		require.Error(t, err, name)
	}
	_, _, _, err := (&TenantRunner{Root: t.TempDir(), Generated: func(Target) ([]GeneratedFile, bool, error) {
		return nil, false, errors.New("loader down")
	}}).source(target("structure"))
	require.ErrorContains(t, err, "loader down", "a generator error is surfaced, not swallowed into a missing app")
}

func TestApply_AGeneratedMigrationRunsOnceAndIsRecorded(t *testing.T) {
	db := scratch(t)
	r := &TenantRunner{Root: t.TempDir(), Generated: gen(GeneratedFile{Filename: "0001_structure.up.sql", SQL: "CREATE TABLE public.gen_t (id int);"})}
	rep, err := r.Apply(context.Background(), db, target("structure"))
	require.NoError(t, err)
	require.True(t, rep.Done)
	require.Equal(t, []string{"0001_structure.up.sql"}, rep.Ran)
	require.Equal(t, 1, count(t, db, `SELECT count(*) FROM pg_tables WHERE tablename='gen_t'`))

	again, err := r.Apply(context.Background(), db, target("structure"))
	require.NoError(t, err)
	require.True(t, again.Done)
	require.Empty(t, again.Ran, "a resumed run applies nothing twice")
}

func TestApply_ADifferentCompilationOfTheSameNameIsDrift(t *testing.T) {
	db := scratch(t)
	first := &TenantRunner{Root: t.TempDir(), Generated: gen(GeneratedFile{Filename: "0001_structure.up.sql", SQL: "CREATE TABLE public.gen_t (id int);"})}
	_, err := first.Apply(context.Background(), db, target("structure"))
	require.NoError(t, err)

	changed := &TenantRunner{Root: t.TempDir(), Generated: gen(GeneratedFile{Filename: "0001_structure.up.sql", SQL: "CREATE TABLE public.gen_t (id int, extra int);"})}
	rep, err := changed.Apply(context.Background(), db, target("structure"))
	require.ErrorIs(t, err, ErrDrift, "a tenant built from an older structure is refused, never silently mixed")
	require.Len(t, rep.Drift, 1)
	require.Equal(t, DriftChanged, rep.Drift[0].Kind)
	require.Empty(t, rep.Ran)
	require.Equal(t, 0, count(t, db, `SELECT count(*) FROM information_schema.columns WHERE table_name='gen_t' AND column_name='extra'`))
}

func TestApply_AFailingGeneratedMigrationRollsBackWhole(t *testing.T) {
	db := scratch(t)
	r := &TenantRunner{Root: t.TempDir(), Generated: gen(GeneratedFile{Filename: "0001_structure.up.sql", SQL: "CREATE TABLE public.half (id int); CREATE TABLE public.half (id int);"})}
	rep, err := r.Apply(context.Background(), db, target("structure"))
	require.Error(t, err)
	require.False(t, rep.Done)
	require.Equal(t, 0, count(t, db, `SELECT count(*) FROM pg_tables WHERE tablename='half'`), "all or nothing")
	require.Equal(t, 0, count(t, db, `SELECT count(*) FROM ivy_meta.migration_log`))
}
