package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type result struct {
	latency time.Duration
	err     error
}

func main() {
	var endpoint string
	var query string
	var clients int
	var count int
	var timeout time.Duration

	flag.StringVar(&endpoint, "url", "http://127.0.0.1:5654/db/query", "benchmark target endpoint")
	flag.StringVar(&query, "q", "select 1", "sql query for q parameter")
	flag.IntVar(&clients, "c", 10, "number of concurrent clients")
	flag.IntVar(&count, "n", 1000, "number of requests per client")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "http client timeout")
	flag.Parse()

	if clients <= 0 {
		fmt.Println("-c must be greater than 0")
		return
	}
	if count <= 0 {
		fmt.Println("-n must be greater than 0")
		return
	}
	total := int64(clients * count)

	target, err := buildURL(endpoint, query)
	if err != nil {
		fmt.Printf("invalid target: %v\n", err)
		return
	}

	transport := &http.Transport{
		MaxIdleConns:        clients * 2,
		MaxIdleConnsPerHost: clients * 2,
		MaxConnsPerHost:     clients * 2,
	}
	client := &http.Client{Transport: transport, Timeout: timeout}

	results := make(chan result, clients*4)

	wg := sync.WaitGroup{}
	start := time.Now()

	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < count; j++ {
				t0 := time.Now()
				err := doRequest(client, target)
				results <- result{latency: time.Since(t0), err: err}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	latencies := make([]time.Duration, 0, total)
	var success int64
	var failed int64
	errorSamples := make([]string, 0, 5)

	for r := range results {
		latencies = append(latencies, r.latency)
		if r.err != nil {
			failed++
			if len(errorSamples) < cap(errorSamples) {
				errorSamples = append(errorSamples, r.err.Error())
			}
			continue
		}
		success++
	}

	totalElapsed := time.Since(start)
	printReport(target, query, clients, count, total, totalElapsed, success, failed, latencies, errorSamples)
}

func buildURL(endpoint string, query string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func doRequest(client *http.Client, target string) error {
	resp, err := client.Get(target)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func printReport(target string, query string, clients int, perClient int, total int64, elapsed time.Duration, success int64, failed int64, latencies []time.Duration, errorSamples []string) {
	sort.Slice(latencies, func(i int, j int) bool {
		return latencies[i] < latencies[j]
	})

	qps := float64(total) / elapsed.Seconds()
	avg := avgLatency(latencies)
	min := percentile(latencies, 0)
	max := percentile(latencies, 100)
	p50 := percentile(latencies, 50)
	p90 := percentile(latencies, 90)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	fmt.Println("=== HTTP Query Benchmark ===")
	fmt.Printf("Target      : %s\n", target)
	fmt.Printf("Query       : %s\n", query)
	fmt.Printf("Clients     : %d\n", clients)
	fmt.Printf("Per Client  : %d\n", perClient)
	fmt.Printf("Requests    : %d\n", total)
	fmt.Printf("Success     : %d\n", success)
	fmt.Printf("Failed      : %d\n", failed)
	fmt.Printf("Elapsed     : %s\n", elapsed)
	fmt.Printf("Throughput  : %.2f req/s\n", qps)
	fmt.Println("--- Latency ---")
	fmt.Printf("min         : %s\n", min)
	fmt.Printf("avg         : %s\n", avg)
	fmt.Printf("p50         : %s\n", p50)
	fmt.Printf("p90         : %s\n", p90)
	fmt.Printf("p95         : %s\n", p95)
	fmt.Printf("p99         : %s\n", p99)
	fmt.Printf("max         : %s\n", max)

	if failed > 0 && len(errorSamples) > 0 {
		fmt.Println("--- Error Samples ---")
		for i, s := range errorSamples {
			fmt.Printf("%d. %s\n", i+1, strings.TrimSpace(s))
		}
	}
}

func avgLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	var total int64
	for _, d := range latencies {
		total += d.Nanoseconds()
	}
	return time.Duration(total / int64(len(latencies)))
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	idx := int((p / 100.0) * float64(len(sorted)-1))
	return sorted[idx]
}
