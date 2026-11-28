
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dhyanio/discache/client"
)

func main() {
	endpoint := flag.String("addr", "127.0.0.1:9080", "Server address")
	mode := flag.String("mode", "get", "Workload: get, set, mixed")
	concurrency := flag.Int("concurrency", 10, "Concurrent clients")
	duration := flag.Duration("duration", 30*time.Second, "Benchmark duration")
	keyCount := flag.Int("keys", 10000, "Number of preloaded keys")
	flag.Parse()

	// Preload keys so GET benchmark measures successful reads.
	fmt.Println("Preloading keys...")

	c, err := client.New(*endpoint, client.Options{})
	if err != nil {
		panic(err)
	}

	for i := 0; i < *keyCount; i++ {
		key := []byte(fmt.Sprintf("bench:%d", i))
		if err := c.Put(nil, key, []byte("benchmark-value"), 0); err != nil {
			panic(err)
		}
	}
	c.Close()

	fmt.Println("Preload complete")
	fmt.Printf("Mode=%s Concurrency=%d Duration=%s\n",
		*mode, *concurrency, *duration)

	var totalOps atomic.Uint64
	var errors atomic.Uint64

	var allLatencies []time.Duration
	var latencyMu sync.Mutex

	stop := make(chan struct{})
	var wg sync.WaitGroup

	start := time.Now()

	for worker := 0; worker < *concurrency; worker++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			conn, err := client.New(*endpoint, client.Options{})
			if err != nil {
				errors.Add(1)
				return
			}
			defer conn.Close()

			rng := rand.New(rand.NewSource(int64(workerID) + time.Now().UnixNano()))
			var latencies []time.Duration
			var localOps uint64
			var localErrors uint64

			for {
				select {
				case <-stop:
					latencyMu.Lock()
					allLatencies = append(allLatencies, latencies...)
					latencyMu.Unlock()

					totalOps.Add(localOps)
					errors.Add(localErrors)
					return
				default:
				}

				keyID := rng.Intn(*keyCount)
				key := []byte(fmt.Sprintf("bench:%d", keyID))

				opStart := time.Now()
				var opErr error

				switch *mode {
				case "get":
					_, opErr = conn.Get(nil, key)

				case "set":
					writeKey := []byte(fmt.Sprintf(
						"write:%d:%d", workerID, localOps,
					))
					opErr = conn.Put(nil, writeKey, []byte("benchmark-value"), 0)

				case "mixed":
					if rng.Intn(100) < 80 {
						_, opErr = conn.Get(nil, key)
					} else {
						writeKey := []byte(fmt.Sprintf(
							"write:%d:%d", workerID, localOps,
						))
						opErr = conn.Put(nil, writeKey, []byte("benchmark-value"), 0)
					}

				default:
					fmt.Printf("Unknown mode: %s\n", *mode)
					return
				}

				latencies = append(latencies, time.Since(opStart))
				localOps++

				if opErr != nil {
					localErrors++
				}
			}
		}(worker)
	}

	time.Sleep(*duration)
	close(stop)
	wg.Wait()

	elapsed := time.Since(start)
	ops := totalOps.Load()

	sort.Slice(allLatencies, func(i, j int) bool {
		return allLatencies[i] < allLatencies[j]
	})

	percentile := func(p float64) time.Duration {
		if len(allLatencies) == 0 {
			return 0
		}
		idx := int(float64(len(allLatencies)-1) * p)
		return allLatencies[idx]
	}

	fmt.Println("\n========== BENCHMARK RESULTS ==========")
	fmt.Printf("Endpoint:     %s\n", *endpoint)
	fmt.Printf("Mode:         %s\n", *mode)
	fmt.Printf("Concurrency:  %d\n", *concurrency)
	fmt.Printf("Duration:     %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Total ops:    %d\n", ops)
	fmt.Printf("Errors:       %d\n", errors.Load())
	fmt.Printf("Throughput:   %.2f ops/sec\n", float64(ops)/elapsed.Seconds())
	fmt.Printf("p50 latency:  %s\n", percentile(0.50))
	fmt.Printf("p95 latency:  %s\n", percentile(0.95))
	fmt.Printf("p99 latency:  %s\n", percentile(0.99))
	fmt.Println("=======================================")
}
