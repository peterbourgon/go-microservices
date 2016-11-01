package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/peterbourgon/go-microservices/addsvc/pkg/endpoints"
	addhttp "github.com/peterbourgon/go-microservices/addsvc/pkg/http"
	"github.com/peterbourgon/go-microservices/addsvc/pkg/service"
)

func main() {
	var (
		addr      = flag.String("addr", ":8080", "HTTP listen address")
		stringsvc = flag.String("stringsvc", "http://localhost:8081", "address of a stringsvc for Concat capitalization")
	)
	flag.Parse()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
		logger = log.NewContext(logger).With("caller", log.DefaultCaller)
	}

	// Our metrics are dependencies, here we create them.
	var ints, chars metrics.Counter
	{
		// Business level metrics.
		ints = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "peterbourgon",
			Subsystem: "addsvc",
			Name:      "integers_summed",
			Help:      "Total count of integers summed via the Sum method.",
		}, []string{})
		chars = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "peterbourgon",
			Subsystem: "addsvc",
			Name:      "characters_concatenated",
			Help:      "Total count of characters concatenated via the Concat method.",
		}, []string{})
	}
	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "peterbourgon",
			Subsystem: "addsvc",
			Name:      "request_duration_seconds",
			Help:      "Request duration in seconds.",
		}, []string{"method", "success"})
	}

	svc := service.New(logger, ints, chars, uppercaseTransform(*stringsvc))
	eps := endpoints.New(svc, logger, duration)
	mux := addhttp.NewHandler(context.Background(), eps, logger)

	logger.Log("transport", "HTTP", "addr", *addr)
	logger.Log("exit", http.ListenAndServe(*addr, mux))
}

func uppercaseTransform(endpoint string) func(string) (string, error) {
	return func(s string) (string, error) {
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = "http://" + endpoint
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(map[string]string{
			"s": s,
		}); err != nil {
			return "", err
		}
		u.Path = "uppercase"
		req, err := http.NewRequest("GET", u.String(), &buf)
		if err != nil {
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		var response struct {
			V string `json:"v"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return "", err
		}
		return response.V, nil
	}
}
