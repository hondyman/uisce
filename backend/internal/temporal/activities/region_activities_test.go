package activities

import (
	"context"
	"strings"
	"testing"

	"github.com/hondyman/uisce/backend/internal/region"
)

const regionActivitiesTestYAML = `
regions:
  us-east-1:
    postgres_host: pg.us-east-1.internal
    postgres_port: 5432
    lakekeeper_url: https://lk.us-east-1.internal:8181
    starrocks_fe_host: sr.us-east-1.internal
    debezium_connect_url: http://connect.us-east-1.internal:8083
`

func newRegionActivitiesForTest(t *testing.T) *RegionActivities {
	t.Helper()
	reg, err := region.ParseRegistry([]byte(regionActivitiesTestYAML))
	if err != nil {
		t.Fatalf("ParseRegistry: %v", err)
	}
	return &RegionActivities{Registry: reg}
}

func TestResolveRegionReturnsConfiguredEndpoints(t *testing.T) {
	a := newRegionActivitiesForTest(t)
	ep, err := a.ResolveRegion(context.Background(), "us-east-1")
	if err != nil {
		t.Fatalf("ResolveRegion: %v", err)
	}
	if ep.PostgresHost != "pg.us-east-1.internal" {
		t.Fatalf("PostgresHost = %q", ep.PostgresHost)
	}
}

func TestResolveRegionFailsClosedForHostileOrUnknownCodes(t *testing.T) {
	a := newRegionActivitiesForTest(t)
	for _, code := range []string{"", "eu-west-1", "US-EAST-1", "us-east-1 ", "us-east-1;drop", "us-east-1\x00"} {
		_, err := a.ResolveRegion(context.Background(), code)
		if err == nil || !strings.Contains(err.Error(), "unknown region") {
			t.Errorf("ResolveRegion(%q) err=%v, want unknown-region rejection", code, err)
		}
	}
}

func TestResolveRegionWithoutRegistryFailsClosed(t *testing.T) {
	a := &RegionActivities{}
	if _, err := a.ResolveRegion(context.Background(), "us-east-1"); err == nil {
		t.Fatal("ResolveRegion without a registry succeeded")
	}
}
