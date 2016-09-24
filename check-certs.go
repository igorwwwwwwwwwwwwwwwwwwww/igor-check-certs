package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// from man sysexits

// ExUsage - The command was used incorrectly, e.g., with the
//           wrong number of arguments, a bad flag, a bad syntax
//           in a parameter, or whatever.
const ExUsage = 64

var hostsFile = flag.String("hosts", "", "path of file containing hostnames to check")
var days = flag.Int("days", 30, "number of days to look into the future")
var concurrency = flag.Int("concurrency", 8, "concurrent checks")

type result struct {
	Hostname string
	Err      error
}

func readHostsFile(hostsFile string) ([]string, error) {
	var hosts []string

	if _, err := os.Stat(hostsFile); os.IsNotExist(err) {
		return hosts, errors.Errorf("provided hosts file %s does not exist", hostsFile)
	}

	f, err := os.Open(hostsFile)
	if err != nil {
		return hosts, errors.Wrap(err, "error opening hosts file")
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		hosts = append(hosts, strings.TrimSpace(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return hosts, errors.Wrap(err, "error reading hosts file")
	}

	return hosts, nil
}

func worker(queue chan string, results chan result) {
	for host := range queue {
		r, err := checkCertificate(host)
		if err != nil {
			r.Err = err
		}
		results <- r
	}
}

func checkCertificate(host string) (result, error) {
	r := result{
		Hostname: host,
	}

	connectHost := host
	if !strings.Contains(host, ":") {
		connectHost = host + ":443"
	}

	conn, err := tls.Dial("tcp", connectHost, &tls.Config{})
	if err != nil {
		return r, errors.Wrap(err, "tls dial")
	}
	conn.Close()

	certExpiry := time.Now().AddDate(0, 0, *days)

	for i, cert := range conn.ConnectionState().PeerCertificates {
		if certExpiry.After(cert.NotAfter) {
			return r, errors.Errorf("cert[%d] %s expires at %v", i, cert.Subject.CommonName, cert.NotAfter)
		}
	}

	return r, nil
}

func main() {
	flag.Parse()

	var hosts []string
	hosts = append(hosts, flag.Args()...)

	if len(*hostsFile) > 0 {
		fileHosts, err := readHostsFile(*hostsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(ExUsage)
		}
		hosts = append(hosts, fileHosts...)
	}

	queue := make(chan string)
	results := make(chan result)

	// start workers
	for i := 0; i < *concurrency; i++ {
		go worker(queue, results)
	}

	// enqueue work
	go func() {
		for _, host := range hosts {
			queue <- host
		}
		close(queue)
	}()

	// consume results
	anyErrors := false
	for i := 0; i < len(hosts); i++ {
		r := <-results
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", r.Hostname, r.Err)
			anyErrors = true
		}
	}

	if anyErrors {
		os.Exit(1)
	}
}
