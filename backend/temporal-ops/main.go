package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hondyman/semlayer/tools/temporal-ops/admin"
)

var (
	formatJSON = flag.Bool("json", false, "output as JSON when possible")
)

func usage() {
	fmt.Println("temporal-ops [describe-queue|list|create|update] args...")
	os.Exit(2)
}

// parseRetention accepts strings like "168h", "7d", "3600s", or plain integer seconds.
func parseRetention(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty retention")
	}
	// try time.ParseDuration first (handles h, m, s)
	if d, err := time.ParseDuration(s); err == nil {
		return int64(d.Seconds()), nil
	}
	// support days suffix 'd' -> hours * 24
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		if v, err := strconv.ParseInt(n, 10, 64); err == nil {
			return v * 24 * 3600, nil
		}
	}
	// fallback: parse as integer seconds
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v, nil
	}
	return 0, fmt.Errorf("unrecognized retention format: %s", s)
}

// printJSON marshals v and writes to stdout.
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// listNamespaces calls the provided client and prints output according to flags.
func listNamespaces(ctx context.Context, c admin.AdminClient) error {
	resp, err := c.ListNamespaces(ctx)
	if err != nil {
		return err
	}
	if *formatJSON {
		return printJSON(resp)
	}
	// fallback: print the raw response (structure may vary by proto version)
	fmt.Printf("%+v\n", resp)
	return nil
}

func describeQueue(ctx context.Context, c admin.AdminClient, namespace, queue string, activity bool) error {
	resp, err := c.DescribeTaskQueue(ctx, namespace, queue, activity)
	if err != nil {
		return err
	}
	if *formatJSON {
		return printJSON(resp)
	}
	// fallback: print the raw response (structure may vary by proto version)
	fmt.Printf("TaskQueue: %s\n%+v\n", queue, resp)
	return nil
}

func createNamespace(ctx context.Context, c admin.AdminClient, ns string, retentionSeconds int64) error {
	if _, err := c.RegisterNamespace(ctx, ns, retentionSeconds); err != nil {
		return err
	}
	if *formatJSON {
		return printJSON(map[string]string{"created": ns})
	}
	fmt.Println("Namespace created:", ns)
	return nil
}

func updateNamespace(ctx context.Context, c admin.AdminClient, ns string, retentionSeconds int64) error {
	if _, err := c.UpdateNamespace(ctx, ns, retentionSeconds); err != nil {
		return err
	}
	if *formatJSON {
		return printJSON(map[string]string{"updated": ns})
	}
	fmt.Println("Namespace updated:", ns)
	return nil
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
	}
	cmd := flag.Arg(0)
	switch cmd {
	case "describe-queue":
		if flag.NArg() < 2 {
			usage()
		}
		queue := flag.Arg(1)
		addr := getEnv("TEMPORAL_GRPC_ENDPOINT", "temporal:7233")
		if flag.NArg() > 2 {
			addr = flag.Arg(2)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := admin.NewClient(ctx, addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to dial admin:", err)
			os.Exit(1)
		}
		defer c.Close()
		if err := describeQueue(ctx, c, "default", queue, false); err != nil {
			fmt.Fprintln(os.Stderr, "DescribeTaskQueue error:", err)
			os.Exit(1)
		}
	case "list":
		addr := getEnv("TEMPORAL_GRPC_ENDPOINT", "temporal:7233")
		if flag.NArg() > 1 {
			addr = flag.Arg(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := admin.NewClient(ctx, addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to dial admin:", err)
			os.Exit(1)
		}
		defer c.Close()
		if err := listNamespaces(ctx, c); err != nil {
			fmt.Fprintln(os.Stderr, "ListNamespaces error:", err)
			os.Exit(1)
		}
	case "create":
		if flag.NArg() < 3 {
			usage()
		}
		ns := flag.Arg(1)
		retentionStr := flag.Arg(2)
		addr := getEnv("TEMPORAL_GRPC_ENDPOINT", "temporal:7233")
		if flag.NArg() > 3 {
			addr = flag.Arg(3)
		}
		seconds, err := parseRetention(retentionStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid retention:", err)
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := admin.NewClient(ctx, addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to dial admin:", err)
			os.Exit(1)
		}
		defer c.Close()
		if err := createNamespace(ctx, c, ns, seconds); err != nil {
			fmt.Fprintln(os.Stderr, "RegisterNamespace error:", err)
			os.Exit(1)
		}
	case "update":
		if flag.NArg() < 3 {
			usage()
		}
		ns := flag.Arg(1)
		retention := flag.Arg(2)
		addr := getEnv("TEMPORAL_GRPC_ENDPOINT", "temporal:7233")
		if flag.NArg() > 3 {
			addr = flag.Arg(3)
		}
		seconds, err := parseRetention(retention)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid retention:", err)
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, err := admin.NewClient(ctx, addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to dial admin:", err)
			os.Exit(1)
		}
		defer c.Close()
		if err := updateNamespace(ctx, c, ns, seconds); err != nil {
			fmt.Fprintln(os.Stderr, "UpdateNamespace error:", err)
			os.Exit(1)
		}
	default:
		usage()
	}
}

// getEnv returns the environment variable or fallback default
func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
