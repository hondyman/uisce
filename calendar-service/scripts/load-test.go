package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL     = "http://localhost:8080/api/v1"
	endpoint    = "/availability/check"
	concurrency = 20
	requests    = 500
)

func main() {
	fmt.Printf("Starting load test on %s%s\n", baseURL, endpoint)
	fmt.Printf("Concurrency: %d, Total Requests: %d\n", concurrency, requests)

	var wg sync.WaitGroup
	startTime := time.Now()

	results := make(chan time.Duration, requests)
	errors := make(chan error, requests)

	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			reqStartTime := time.Now()
			// Mock request for now - in production use real auth token
			resp, err := http.Get(baseURL + endpoint + "?profile=primary&tenant=TEST_TENANT")
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				errors <- fmt.Errorf("status: %d", resp.StatusCode)
				return
			}

			results <- time.Since(reqStartTime)
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	duration := time.Since(startTime)

	var totalDuration time.Duration
	count := 0
	for d := range results {
		totalDuration += d
		count++
	}

	fmt.Printf("\n--- Load Test Results ---\n")
	fmt.Printf("Total Time: %v\n", duration)
	fmt.Printf("Successes: %d\n", count)
	fmt.Printf("Failures: %d\n", len(errors))
	if count > 0 {
		fmt.Printf("Average Latency: %v\n", totalDuration/time.Duration(count))
		fmt.Printf("Requests/sec: %.2f\n", float64(count)/duration.Seconds())
	}
}
