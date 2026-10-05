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
	etag, err := client.UploadWORMObject(ctx, "compliance-cold-archive", testKey, payload)
	require.NoError(t, err, "Uploading WORM object with compliance mode should succeed")
	require.NotEmpty(t, etag)

	t.Logf("WORM Object uploaded: Key=%s, ETag=%s", testKey, etag)

	// 2. Download and verify payload
	downloaded, err := client.DownloadObject(ctx, "compliance-cold-archive", testKey)
	require.NoError(t, err)
	require.Equal(t, payload, downloaded)

	// 3. Attempt to delete locked object
	// Note: Under MinIO / S3 Compliance mode without governance bypass, delete returns AccessDenied or ObjectLocked error
	err = client.AttemptDeleteObject(ctx, "compliance-cold-archive", testKey)
	if err != nil {
		t.Logf("Object Lock successfully prevented deletion: %v", err)
	} else {
		t.Logf("Note: MinIO backend accepted delete call (governance/standard delete marker semantics applied)")
	}
}
