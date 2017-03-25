package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	addendpoint "github.com/peterbourgon/go-microservices/addsvc/pkg/endpoint"
	addhttp "github.com/peterbourgon/go-microservices/addsvc/pkg/http"
	addservice "github.com/peterbourgon/go-microservices/addsvc/pkg/service"
)

func main() {
	var (
		httpAddr       = flag.String("http-addr", ":8080", "HTTP listen address")
		zipkinURL      = flag.String("zipkin-url", "", "Zipkin collector URL e.g. http://localhost:9411/api/v1/spans")
		postprocessURL = flag.String("postprocess-url", "", "URL for postprocessing results of Concat e.g. http://stringsvc.default.svc.cluster.local/uppercase")
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

	postprocess := func(s string) string { return s } // default no-op
	if *postprocessURL != "" {
		postprocess = func(s string) string {
			body, err := json.Marshal(map[string]string{"s": s})
			if err != nil {
				logger.Log("during", "postprocess", "err", err)
				return s
			}

			req, err := http.NewRequest("GET", *postprocessURL, bytes.NewReader(body))
			if err != nil {
				logger.Log("during", "postprocess", "err", err)
				return s
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.Log("during", "postprocess", "err", err)
				return s
			}
			defer resp.Body.Close()

			var response struct {
				V string `json:"v"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				logger.Log("during", "postprocess", "err", err)
				return s
			}

			return response.V
		}
	}

	svc := addservice.New(postprocess, logger, ints, chars)
	eps := addendpoint.New(svc, logger, duration, tracer)
	mux := addhttp.NewHandler(context.Background(), eps, logger, tracer)

	logger.Log("transport", "HTTP", "addr", *httpAddr)
	logger.Log("exit", http.ListenAndServe(*httpAddr, mux))
}
