package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/peterbourgon/go-microservices/pkg/endpoints"
	addhttp "github.com/peterbourgon/go-microservices/pkg/http"
	"github.com/peterbourgon/go-microservices/pkg/service"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
		logger = log.NewContext(logger).With("caller", log.DefaultCaller)
	}

	var trace stdopentracing.Tracer
	{
		trace = stdopentracing.GlobalTracer() // no-op
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

	svc := service.New(logger, ints, chars)
	eps := endpoints.New(svc, logger, duration, trace)
	mux := addhttp.NewHandler(context.Background(), eps, logger, trace)

	logger.Log("transport", "HTTP", "addr", *addr)
	logger.Log("exit", http.ListenAndServe(*addr, mux))
}
