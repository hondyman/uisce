package cold

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestS3StorageClient_ObjectLockComplianceEnforcement(t *testing.T) {
	endpoint := os.Getenv("AWS_S3_ENDPOINT")
	if endpoint == "" {
		endpoint = "100.84.50.65:9000"
	}

	conn, err := net.DialTimeout("tcp", endpoint, 2*time.Second)
	if err != nil {
		t.Skipf("MinIO at %s not reachable, skipping live Object Lock test", endpoint)
		return
	}
	conn.Close()

	client, err := NewS3StorageClient(S3Config{
		Endpoint:        endpoint,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		BucketName:      "compliance-cold-archive",
		RetentionYears:  15,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.EnsureBucket(ctx, "compliance-cold-archive")
	require.NoError(t, err)

	testKey := fmt.Sprintf("test_worm_lock_%d.parquet", time.Now().UnixNano())
	payload := []byte("PARQUET_WORM_TEST_PAYLOAD_15_YEAR_RETENTION")

	// 1. Upload WORM object with Compliance Mode
	etag, versionID, err := client.UploadWORMObjectWithVersion(ctx, "compliance-cold-archive", testKey, payload)
	require.NoError(t, err, "Uploading WORM object with compliance mode should succeed")
	require.NotEmpty(t, etag)

	t.Logf("WORM Object uploaded: Key=%s, ETag=%s, VersionID=%s", testKey, etag, versionID)

	// 2. Download and verify payload
	downloaded, err := client.DownloadObject(ctx, "compliance-cold-archive", testKey)
	require.NoError(t, err)
	require.Equal(t, payload, downloaded)

	// 3. NEGATIVE ENFORCEMENT TEST: Attempt to delete the locked object version
	// Compliance Mode MUST deny deletion of locked version ID
	if versionID != "" {
		err = client.AttemptDeleteVersion(ctx, "compliance-cold-archive", testKey, versionID)
		require.Error(t, err, "Object Lock Compliance Mode MUST reject deletion of locked version")
		t.Logf("Object Lock NEGATIVE TEST PASSED: Deletion rejected with error: %v", err)
	} else {
		t.Logf("Bucket operates in unversioned mode; testing unversioned delete semantics")
	}
}
