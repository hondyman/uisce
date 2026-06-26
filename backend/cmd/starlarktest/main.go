package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/services"
)

type testCase struct {
	Record any    `json:"record"`
	Data   any    `json:"data"`
	Ctx    any    `json:"ctx"`
	Expect *bool  `json:"expect"`
	Name   string `json:"name"`
}

func main() {
	filePath := flag.String("file", "", "Path to .star file")
	flag.Parse()

	if strings.TrimSpace(*filePath) == "" {
		fmt.Fprintln(os.Stderr, "starlarktest: -file is required")
		os.Exit(2)
	}

	src, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "starlarktest: read file: %v\n", err)
		os.Exit(2)
	}

	cases, err := extractTestCases(string(src))
	if err != nil {
		fmt.Fprintf(os.Stderr, "starlarktest: parse testcases: %v\n", err)
		os.Exit(2)
	}
	if len(cases) == 0 {
		fmt.Fprintln(os.Stderr, "starlarktest: no testcases found (add comment lines like: # {\"record\": {...}, \"expect\": true})")
		os.Exit(2)
	}

	engine := services.NewStarlarkEngine(nil)
	var failed int

	for i, tc := range cases {
		name := tc.Name
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("case_%d", i+1)
		}

		dataMap, ok := testcaseInputToMap(tc)
		if !ok {
			failed++
			fmt.Fprintf(os.Stderr, "FAIL %s: testcase must include record/data/ctx object\n", name)
			continue
		}
		if tc.Expect == nil {
			failed++
			fmt.Fprintf(os.Stderr, "FAIL %s: missing expect (true/false)\n", name)
			continue
		}

		res, _ := engine.EvaluateUserRule(context.Background(), string(src), dataMap)
		if res == nil {
			failed++
			fmt.Fprintf(os.Stderr, "FAIL %s: got nil result\n", name)
			continue
		}

		if res.IsValid != *tc.Expect {
			failed++
			fmt.Fprintf(os.Stderr, "FAIL %s: expect=%v got=%v message=%q\n", name, *tc.Expect, res.IsValid, res.Message)
			continue
		}

		fmt.Fprintf(os.Stdout, "PASS %s\n", name)
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "%d/%d failed\n", failed, len(cases))
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "ok (%d cases)\n", len(cases))
}

func extractTestCases(src string) ([]testCase, error) {
	// Convention: any comment line that begins with `# {` is treated as a testcase JSON object.
	// Example:
	//   # {"name":"happy","record":{"account":{"aum":150}},"expect":true}
	var cases []testCase
	scanner := bufio.NewScanner(strings.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if !strings.HasPrefix(payload, "{") {
			continue
		}
		var tc testCase
		if err := json.Unmarshal([]byte(payload), &tc); err != nil {
			return nil, fmt.Errorf("invalid testcase json: %w", err)
		}
		cases = append(cases, tc)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cases, nil
}

func testcaseInputToMap(tc testCase) (map[string]any, bool) {
	// Prefer record/data; if ctx is provided, treat it as the root input record.
	var v any
	switch {
	case tc.Record != nil:
		v = tc.Record
	case tc.Data != nil:
		v = tc.Data
	case tc.Ctx != nil:
		v = tc.Ctx
	default:
		return nil, false
	}

	m, ok := v.(map[string]any)
	if ok {
		return m, true
	}

	// json.Unmarshal into interface{} can yield map[string]interface{}.
	m2, ok := v.(map[string]interface{})
	if ok {
		out := make(map[string]any, len(m2))
		for k, vv := range m2 {
			out[k] = vv
		}
		return out, true
	}
	return nil, false
}
