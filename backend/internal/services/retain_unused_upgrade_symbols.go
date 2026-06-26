package services

// retain_unused_upgrade_symbols.go
// Retention shim to reference unused fields and types in upgrade_service.go so
// staticcheck does not report them as unused for intentionally-kept lifecycle
// and runtime structures.
func init() {
	// Reference types to mark fields as used
	var uls UpgradeLifecycleService
	_ = uls.activeModelVersion
	_ = uls.versionsMap
	_ = uls.deprecationMaps
	_ = uls.schemaChanges
	_ = uls.validationReports
	_ = uls.shadowRuns
	_ = uls.preAggRebuilds

	var urs UpgradeRuntimeService
	_ = &urs.mu // Use address to avoid copying mutex
	_ = urs.versions
	_ = urs.order
	_ = urs.active
	_ = urs.previous
	_ = urs.preview
	_ = urs.canary
	_ = urs.slo
	_ = urs.notifications
	_ = urs.wsHub
	_ = urs.activeModelVersion
	_ = urs.versionsMap
	_ = urs.deprecationMaps
	_ = urs.schemaChanges
	_ = urs.validationReports
	_ = urs.shadowRuns
	_ = urs.preAggRebuilds

	// Reference a constructor and method values
	_ = NewUpgradeRuntimeService
	_ = urs.ListVersions
}
