package graphql

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/vektah/gqlparser/v2/ast"
)

// Minimal unmarshal/marshal helpers for custom scalars used in generated code.
// These mirror the helpers gqlgen normally generates; providing them here resolves
// transient generation inconsistencies and keeps the code compile-able.

func (ec *executionContext) unmarshalInputTimestamp(_ context.Context, obj any) (time.Time, error) {
	if obj == nil {
		return time.Time{}, fmt.Errorf("timestamp cannot be null")
	}
	switch v := obj.(type) {
	case string:
		// try RFC3339
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, nil
		}
		// try unix seconds
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(i, 0).UTC(), nil
		}
	case float64:
		return time.Unix(int64(v), 0).UTC(), nil
	case int:
		return time.Unix(int64(v), 0).UTC(), nil
	case int64:
		return time.Unix(v, 0).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp value: %v", obj)
}

func (ec *executionContext) _Timestamp(_ context.Context, _ ast.SelectionSet, v *time.Time) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(v.Format(time.RFC3339)))
	})
}

func (ec *executionContext) unmarshalInputUUID(_ context.Context, obj any) (uuid.UUID, error) {
	if obj == nil {
		return uuid.Nil, fmt.Errorf("uuid cannot be null")
	}
	switch v := obj.(type) {
	case string:
		u, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid UUID: %w", err)
		}
		return u, nil
	}
	return uuid.Nil, fmt.Errorf("invalid uuid value: %v", obj)
}

func (ec *executionContext) _UUID(_ context.Context, _ ast.SelectionSet, v *uuid.UUID) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(v.String()))
	})
}
