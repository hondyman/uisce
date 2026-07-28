package upgrade

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpgradePackageManifest_ComputeChecksum(t *testing.T) {
	manifest := CreateSampleManifest("v1.3.0")
	assert.NotEmpty(t, manifest.Checksum, "Checksum must be generated for upgrade package")
	assert.Equal(t, 2, len(manifest.CoreDeltas))
	assert.Equal(t, 2, len(manifest.SchemaScripts))
}

func TestImpactEngine_RunPreFlightSimulation(t *testing.T) {
	manifest := CreateSampleManifest("v1.3.0")
	engine := NewImpactEngine(nil)

	report, err := engine.RunPreFlightSimulation(context.Background(), manifest)
	assert.NoError(t, err)
	assert.Equal(t, manifest.PackageID, report.PackageID)
	assert.Equal(t, 2, report.CoreDeltasCount)
	assert.NotEmpty(t, report.StorageImpact.CitusDistributedDDLs)
	assert.NotEmpty(t, report.StorageImpact.IcebergEvolutions)
}

func TestDistributedExecutor_ExecuteGlobalDeployment(t *testing.T) {
	manifest := CreateSampleManifest("v1.3.0")
	executor := NewDistributedExecutor(nil)

	res, err := executor.ExecuteGlobalDeployment(context.Background(), manifest)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", res.OverallStatus)
	assert.Len(t, res.ExecutionSteps, 4, "Must execute Citus, Cache, Polyglot, and Regional Canary steps")
	assert.Equal(t, 4, len(res.RegionsDeployed))
}
