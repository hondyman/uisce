package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hondyman/uisce/backend/internal/compliance/cold"
)

func main() {
	endpoint := flag.String("endpoint", os.Getenv("AWS_S3_ENDPOINT"), "S3 / MinIO endpoint host:port")
	accessKey := flag.String("access-key", os.Getenv("AWS_ACCESS_KEY_ID"), "S3 access key")
	secretKey := flag.String("secret-key", os.Getenv("AWS_SECRET_ACCESS_KEY"), "S3 secret key")
	bucket := flag.String("bucket", "compliance-cold-archive", "S3 archive bucket")
	key := flag.String("key", "", "S3 object key to verify")
	manifestRoot := flag.String("manifest-root", "", "Expected Merkle Root from signed manifest")
	localFile := flag.String("local-file", "", "Path to local Parquet file (offline audit mode)")
	outputJSON := flag.Bool("json", true, "Output report as structured JSON")
	flag.Parse()

	if *localFile == "" && (*key == "" || *manifestRoot == "") {
		log.Fatalf("Usage: compliance-verify --bucket <bucket> --key <key> --manifest-root <root_hex> OR --local-file <path> --manifest-root <root_hex>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var parquetBytes []byte
	var err error

	if *localFile != "" {
		parquetBytes, err = os.ReadFile(*localFile)
		if err != nil {
			log.Fatalf("Failed to read local file %s: %v", *localFile, err)
		}
	} else {
		if *endpoint == "" {
			*endpoint = "100.84.50.65:9000"
		}
		if *accessKey == "" {
			*accessKey = "minioadmin"
		}
		if *secretKey == "" {
			*secretKey = "minioadmin"
		}

		s3Client, err := cold.NewS3StorageClient(cold.S3Config{
			Endpoint:        *endpoint,
			AccessKeyID:     *accessKey,
			SecretAccessKey: *secretKey,
			UseSSL:          false,
			BucketName:      *bucket,
		})
		if err != nil {
			log.Fatalf("Failed to initialize S3 client: %v", err)
		}

		parquetBytes, err = s3Client.DownloadObject(ctx, *bucket, *key)
		if err != nil {
			log.Fatalf("Failed to download object from S3 (%s/%s): %v", *bucket, *key, err)
		}
	}

	verifier := cold.NewArchiveVerifier()
	report, err := verifier.VerifyParquetSlice(ctx, parquetBytes, *manifestRoot)
	if err != nil {
		log.Fatalf("Verification error: %v", err)
	}
	report.S3Bucket = *bucket
	report.S3Key = *key

	if *outputJSON {
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Printf("=== COMPLIANCE AUDIT CERTIFICATE ===\n")
		fmt.Printf("Tenant ID       : %s\n", report.TenantID)
		fmt.Printf("LSN Range       : [%d, %d]\n", report.StartLSN, report.EndLSN)
		fmt.Printf("Record Count    : %d\n", report.RecordCount)
		fmt.Printf("File Size Bytes : %d\n", report.FileSizeBytes)
		fmt.Printf("SHA256 Checksum : %s\n", report.SHA256Checksum)
		fmt.Printf("Manifest Root   : %s\n", report.ManifestRoot)
		fmt.Printf("Computed Root   : %s\n", report.ComputedRoot)
		fmt.Printf("Merkle Match    : %v\n", report.RootMatch)
		fmt.Printf("Inclusion Proofs: %v\n", report.InclusionProofs)
		fmt.Printf("Audit Status    : %s\n", report.AuditStatus)
		fmt.Printf("====================================\n")
	}

	if !report.RootMatch || !report.InclusionProofs {
		os.Exit(2)
	}
}
