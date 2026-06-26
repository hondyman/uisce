package boresolver_test

import (
	"errors"
	"testing"

	br "github.com/hondyman/semlayer/backend/internal/boresolver"
)

type fakeRepo struct {
	def *br.BODefinition
	err error
}

func (f *fakeRepo) GetBODefinition(boID string) (*br.BODefinition, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.def, nil
}

func (f *fakeRepo) GetBOByTechnicalName(tenantID, datasourceID, technicalName string) (*br.BODefinition, error) {
	return f.def, f.err
}

func TestValidateSelectedFields_AllValid(t *testing.T) {
	repo := &fakeRepo{def: &br.BODefinition{Fields: []br.BOField{{ID: "a"}, {ID: "b"}}}}
	invalid, err := br.ValidateSelectedFields(repo, "bo1", []string{"a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invalid) != 0 {
		t.Fatalf("expected no invalid ids, got: %v", invalid)
	}
}

func TestValidateSelectedFields_Invalids(t *testing.T) {
	repo := &fakeRepo{def: &br.BODefinition{Fields: []br.BOField{{ID: "a"}, {ID: "b"}}}}
	invalid, err := br.ValidateSelectedFields(repo, "bo1", []string{"x", "a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invalid) != 1 || invalid[0] != "x" {
		t.Fatalf("expected invalid [x], got: %v", invalid)
	}
}

func TestValidateSelectedFields_ErrorFromRepo(t *testing.T) {
	repo := &fakeRepo{err: errors.New("not found")}
	_, err := br.ValidateSelectedFields(repo, "bo1", []string{"a"})
	if err == nil {
		t.Fatalf("expected error from repo, got nil")
	}
}
