package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	defaultPort    = 5353
	defaultTimeout = 2 * time.Second
)

var (
	target   string
	port     string
	count    int
	interval time.Duration
	verbose  bool
)

func init() {
	flag.StringVar(&target, "target", "", "Target hostname or IP address")
	flag.StringVar(&port, "port", fmt.Sprintf("%d", defaultPort), "Target port")
	flag.IntVar(&count, "count", 4, "Number of pings")
	flag.DurationVar(&interval, "interval", 1*time.Second, "Time between pings")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&verbose, "help", false, "Show help message")
}

func main() {
	flag.Parse()

	if verbose {
		printUsage()
		os.Exit(0)
	}

	if target == "" {
		printUsage()
		os.Exit(1)
	}

	if count < 1 || count > 10 {
		fmt.Fprintf(os.Stderr, "Count must be between 1 and 10\n")
		os.Exit(1)
	}

	if interval < 100*time.Millisecond {
		fmt.Fprintf(os.Stderr, "Interval must be at least 100ms\n")
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Pinging %s on port %s (%d times, %s interval)...\n\n",
			target, port, count, interval)
	}

	successful := 0
	latencies := make([]time.Duration, 0, count)

	for i := 0; i < count; i++ {
		start := time.Now()

		conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%s", target, port), defaultTimeout)
		if err != nil {
			if verbose {
				fmt.Printf("[%d] ERROR: %v\n", i+1, err)
			}
			time.Sleep(interval)
			continue
		}
		defer conn.Close()

		conn.SetDeadline(time.Now().Add(defaultTimeout))
		_, err = conn.Write([]byte("PING"))
		if err != nil {
			if verbose {
				fmt.Printf("[%d] ERROR: %v\n", i+1, err)
			}
			time.Sleep(interval)
			continue
		}

		conn.SetDeadline(time.Now().Add(defaultTimeout))
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			if verbose {
				fmt.Printf("[%d] ERROR: %v\n", i+1, err)
			}
		} else if strings.HasPrefix(string(buf), "PONG") {
			latency := time.Since(start)
			latencies = append(latencies, latency)
			successful++
			if verbose {
				fmt.Printf("[%d] PONG from %s (latency: %v)\n", i+1, target, latency)
			}
		} else {
			if verbose {
				fmt.Printf("[%d] Unexpected response: %s\n", i+1, string(buf))
			}
		}

		time.Sleep(interval)
	}

	if successful > 0 {
		var total time.Duration
		for _, l := range latencies {
			if l > 0 {
				total += l
			}
		}
		avgLatency := total / time.Duration(successful)

		fmt.Printf("\n=== Results ===\n")
		fmt.Printf("Target:    %s:%s\n", target, port)
		fmt.Printf("Success:   %d/%d\n", successful, count)
		fmt.Printf("Latency:   %v (avg over %d successes)\n", avgLatency, successful)

		if successful == count {
			fmt.Println("Status:     UP")
		} else {
			lossRate := float64(count-successful) / float64(count) * 100
			fmt.Printf("Status:     PARTIAL (%.0f%% loss)\n", lossRate)
		}
	} else {
		fmt.Printf("\n=== Results ===\n")
		fmt.Printf("Target:  %s:%s\n", target, port)
		fmt.Printf("Success:   0/%d\n", count)
		fmt.Println("Status:     DOWN")
	}

	exitCode := 0
	if successful > 0 {
		exitCode = 0
	} else {
		exitCode = 1
	}

	os.Exit(exitCode)
}

func printUsage() {
	fmt.Println("nss-ping - Network ping utility for NSS daemon")
	fmt.Println()
	fmt.Println("WHAT:")
	fmt.Println("  Simple network ping tool for testing daemon connectivity.")
	fmt.Println("  Sends UDP ping requests to check if daemon is responding.")
	fmt.Println()
	fmt.Println("WHY:")
	fmt.Println("  • Quick connectivity test without starting full daemon")
	fmt.Println("  • Verify NSS daemon is running on other nodes")
	fmt.Println("  • Test network latency and packet loss")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  nss-ping [options] <target>")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -target <addr>   Target hostname or IP (required)")
	fmt.Println("  -port <port>      Target port (default: 5353)")
	fmt.Println("  -count <n>        Number of pings (default: 4)")
	fmt.Println("  -interval <dur>    Time between pings (default: 1s)")
	fmt.Println("  -v                Verbose output")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  nss-ping webserver")
	fmt.Println("  nss-ping 192.168.1.10")
	fmt.Println("  nss-ping -count 10 -interval 500ms mailserver")
	fmt.Println("  nss-ping -v 192.168.1.10")
	fmt.Println()
	fmt.Println("RELATED:")
	fmt.Println("  nss-daemon       The main daemon")
	fmt.Println("  nss-query         Query daemon for hosts and services")
	fmt.Println("  nss-status         Status and records viewer")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/offline-lab/disco")
}
