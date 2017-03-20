package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/peterbourgon/go-microservices/addsvc/pkg/endpoints"
	addhttp "github.com/peterbourgon/go-microservices/addsvc/pkg/http"
	"github.com/peterbourgon/go-microservices/addsvc/pkg/service"
)

func main() {
	var (
		httpAddr  = flag.String("http-addr", ":8080", "HTTP listen address")
		zipkinURL = flag.String("zipkin-url", "", "Zipkin collector URL e.g. http://localhost:9411/api/v1/spans")
	)
	flag.Parse()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	var tracer stdopentracing.Tracer
	{
		if *zipkinURL != "" {
			logger.Log("zipkin", *zipkinURL)
			collector, err := zipkin.NewHTTPCollector(*zipkinURL)
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
			defer collector.Close()
			var (
				debug       = false
				hostPort    = "localhost:80"
				serviceName = "addsvc"
			)
			tracer, err = zipkin.NewTracer(zipkin.NewRecorder(
				collector, debug, hostPort, serviceName,
			))
			if err != nil {
				logger.Log("err", err)
				os.Exit(1)
			}
		} else {
			tracer = stdopentracing.GlobalTracer() // no-op
		}
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
	eps := endpoints.New(svc, logger, duration, tracer)
	mux := addhttp.NewHandler(context.Background(), eps, logger, tracer)

	logger.Log("transport", "HTTP", "addr", *httpAddr)
	logger.Log("exit", http.ListenAndServe(*httpAddr, mux))
}
