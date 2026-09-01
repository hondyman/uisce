package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/models"
)

func TestPermissionPriority(t *testing.T) {
	tests := []struct {
		perm     string
		expected int
	}{
		{PermissionWrite, 1},
		{PermissionRead, 2},
		{PermissionMask, 3},
		{PermissionNone, 4},
		{"unknown", 5},
	}

	for _, tt := range tests {
		t.Run(tt.perm, func(t *testing.T) {
			assert.Equal(t, tt.expected, permissionPriority(tt.perm))
		})
	}
}

func TestEntitlementsService_ForUser_GlobalAdmin(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	secCtx := &Context{
		TenantID:     "tenant-1",
		UserID:      "admin-user",
		Roles:       []string{"admin"},
		IsGlobalAdmin: true,
	}

	result, err := svc.ForUser(context.Background(), secCtx)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEntitlementsService_ForUser_EmptyRoles(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	secCtx := &Context{
		TenantID:   "tenant-1",
		UserID:    "user-1",
		Roles:     []string{},
	}

	result, err := svc.ForUser(context.Background(), secCtx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Entitlements)
	assert.Empty(t, result.HiddenBOs)
}

func TestEntitlementsService_CanWrite_GlobalAdmin(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	secCtx := &Context{
		TenantID:     "tenant-1",
		UserID:      "admin-user",
		Roles:       []string{"admin"},
		IsGlobalAdmin: true,
	}

	canWrite, err := svc.CanWrite(context.Background(), secCtx, "any-bo-id")
	require.NoError(t, err)
	assert.True(t, canWrite)
}

func TestEntitlementsService_CanWrite_ClosedByDefault(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	secCtx := &Context{
		TenantID:     "tenant-1",
		UserID:      "user-1",
		Roles:       []string{},
		IsGlobalAdmin: false,
	}

	canWrite, err := svc.CanWrite(context.Background(), secCtx, "any-bo-id")
	require.NoError(t, err)
	assert.False(t, canWrite, "closed-by-default: no entry means deny")
}

func TestEntitlementsService_CanRunAI_DelegatesToCanWrite(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	secCtx := &Context{
		TenantID:     "tenant-1",
		UserID:      "admin-user",
		Roles:       []string{"admin"},
		IsGlobalAdmin: true,
	}

	canRun, err := svc.CanRunAI(context.Background(), secCtx, "any-bo-id")
	require.NoError(t, err)
	assert.True(t, canRun)
}

func TestEntitlementsService_CacheKey_SortsRoles(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)

	key1 := svc.cacheKey("tenant-1", "instance-1", "user-1", []string{"admin", "editor", "viewer"})
	key2 := svc.cacheKey("tenant-1", "instance-1", "user-1", []string{"viewer", "admin", "editor"})
	key3 := svc.cacheKey("tenant-1", "instance-1", "user-1", []string{"admin", "editor"})

	assert.Equal(t, key1, key2, "same roles in different order should produce same key")
	assert.NotEqual(t, key1, key3, "different roles should produce different key")
}

func TestEntitlementsService_InvalidateAll(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)

	svc.InvalidateAll("tenant-1")
	svc.InvalidateAll("tenant-2")
}

func TestEntitlementsService_Invalidate(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)

	svc.Invalidate("tenant-1", "user-1")
	svc.Invalidate("tenant-1", "user-2")
}

func TestFilterFields_AllVisible(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	fields := []*models.FieldDefinition{
		{Key: "f1", Name: "Field1"},
		{Key: "f2", Name: "Field2"},
	}

	entitlements := &EntitlementsResult{
		Entitlements:    map[EntitlementKey]string{},
		MaskingPatterns: map[EntitlementKey]string{},
		HiddenBOs:       map[string]struct{}{},
		MaskedFields:    map[EntitlementKey]struct{}{},
	}

	visible, hidden, masked := svc.FilterFields("bo-1", fields, entitlements)
	assert.Len(t, visible, 2)
	assert.Empty(t, hidden)
	assert.Empty(t, masked)
}

func TestFilterFields_NoneHidesField(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	fields := []*models.FieldDefinition{
		{Key: "f1", Name: "Field1"},
		{Key: "f2", Name: "Field2"},
	}

	entitlements := &EntitlementsResult{
		Entitlements: map[EntitlementKey]string{
			{ResourceID: "bo-1", FieldName: "Field1"}: PermissionNone,
		},
		MaskingPatterns: map[EntitlementKey]string{},
		HiddenBOs:       map[string]struct{}{},
		MaskedFields:    map[EntitlementKey]struct{}{},
	}

	visible, hidden, _ := svc.FilterFields("bo-1", fields, entitlements)
	assert.Len(t, visible, 1)
	assert.Equal(t, "Field2", visible[0].Name)
	assert.Len(t, hidden, 1)
	assert.Equal(t, "Field1", hidden[0])
}

func TestFilterFields_MaskMasksField(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	fields := []*models.FieldDefinition{
		{Key: "f1", Name: "SSN"},
		{Key: "f2", Name: "Name"},
	}

	entitlements := &EntitlementsResult{
		Entitlements: map[EntitlementKey]string{
			{ResourceID: "bo-1", FieldName: "SSN"}: PermissionMask,
		},
		MaskingPatterns: map[EntitlementKey]string{
			{ResourceID: "bo-1", FieldName: "SSN"}: "XXX-XX-####",
		},
		HiddenBOs:    map[string]struct{}{},
		MaskedFields: map[EntitlementKey]struct{}{},
	}

	visible, hidden, masked := svc.FilterFields("bo-1", fields, entitlements)
	assert.Len(t, visible, 2)
	assert.Empty(t, hidden)
	assert.NotNil(t, masked)
	pattern, ok := masked["SSN"]
	assert.True(t, ok)
	assert.Equal(t, "XXX-XX-####", pattern)
}

func TestFilterFields_NilEntitlements(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	fields := []*models.FieldDefinition{
		{Key: "f1", Name: "Field1"},
	}

	visible, hidden, masked := svc.FilterFields("bo-1", fields, nil)
	assert.Len(t, visible, 1)
	assert.Empty(t, hidden)
	assert.Nil(t, masked)
}

func TestFilterFields_ReadWriteAllowsField(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	fields := []*models.FieldDefinition{
		{Key: "f1", Name: "Field1"},
		{Key: "f2", Name: "Field2"},
	}

	entitlements := &EntitlementsResult{
		Entitlements: map[EntitlementKey]string{
			{ResourceID: "bo-1", FieldName: "Field1"}: PermissionWrite,
			{ResourceID: "bo-1", FieldName: "Field2"}: PermissionRead,
		},
		MaskingPatterns: map[EntitlementKey]string{},
		HiddenBOs:       map[string]struct{}{},
		MaskedFields:    map[EntitlementKey]struct{}{},
	}

	visible, hidden, _ := svc.FilterFields("bo-1", fields, entitlements)
	assert.Len(t, visible, 2)
	assert.Empty(t, hidden)
}

func TestFilterFields_UsesKeyAsFallback(t *testing.T) {
	svc := NewEntitlementsService(nil, 100, 5*time.Minute)
	fields := []*models.FieldDefinition{
		{Key: "ssn_field", Name: ""},
	}

	entitlements := &EntitlementsResult{
		Entitlements: map[EntitlementKey]string{
			{ResourceID: "bo-1", FieldName: "ssn_field"}: PermissionMask,
		},
		MaskingPatterns: map[EntitlementKey]string{
			{ResourceID: "bo-1", FieldName: "ssn_field"}: "***",
		},
		HiddenBOs:    map[string]struct{}{},
		MaskedFields: map[EntitlementKey]struct{}{},
	}

	visible, _, _ := svc.FilterFields("bo-1", fields, entitlements)
	assert.Len(t, visible, 1)
	assert.True(t, visible[0].Masked)
}
