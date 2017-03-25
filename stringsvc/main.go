package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/tracing/opentracing"
	httptransport "github.com/go-kit/kit/transport/http"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Configuration from the environment.
	var (
		httpAddr  = flag.String("http-addr", ":8081", "HTTP listen address")
		zipkinURL = flag.String("zipkin-url", "", "Zipkin collector URL e.g. http://localhost:9411/api/v1/spans")
	)
	flag.Parse()

	// Logging domain.
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stdout)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	logger.Log("msg", "hello")
	defer logger.Log("msg", "goodbye")

	// Tracer.
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

	// Metrics domain.
	var requestCount metrics.Counter
	var requestLatency, countResult metrics.Histogram
	{
		requestCount = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "peterbourgon",
			Subsystem: "stringsvc",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method", "error"})
		requestLatency = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "peterbourgon",
			Subsystem: "stringsvc",
			Name:      "request_latency_seconds",
			Help:      "Request duration in seconds.",
		}, []string{"method", "error"})
		countResult = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "peterbourgon",
			Subsystem: "stringsvc",
			Name:      "count_result",
			Help:      "The result of each count method.",
		}, []string{}) // no fields here
	}

	// Construct the service.
	mux := makeServeMux(logger, requestCount, requestLatency, countResult, tracer)
	mux.Handle("/metrics", promhttp.Handler())

	// Go!
	logger.Log("transport", "HTTP", "addr", *httpAddr)
	logger.Log("exit", http.ListenAndServe(*httpAddr, mux))
}

func makeServeMux(
	logger log.Logger,
	requestCount metrics.Counter,
	requestLatency, countResult metrics.Histogram,
	tracer stdopentracing.Tracer,
) *http.ServeMux {
	// Business domain.
	var svc StringService
	{
		svc = stringService{}
		svc = loggingMiddleware{logger, svc}
		svc = instrumentingMiddleware{requestCount, requestLatency, countResult, svc}
	}

	// Endpoint domain.
	var uppercaseEndpoint endpoint.Endpoint
	{
		uppercaseEndpoint = makeUppercaseEndpoint(svc)
		uppercaseEndpoint = opentracing.TraceServer(tracer, "Uppercase")(uppercaseEndpoint)
	}
	var countEndpoint endpoint.Endpoint
	{
		countEndpoint = makeCountEndpoint(svc)
		countEndpoint = opentracing.TraceServer(tracer, "Count")(countEndpoint)
	}

	// Transport domain.
	mux := http.NewServeMux()
	{
		mux.Handle("/uppercase", httptransport.NewServer(
			uppercaseEndpoint,
			decodeUppercaseRequest,
			encodeResponse,
			httptransport.ServerBefore(opentracing.FromHTTPRequest(tracer, "Uppercase", logger)),
		))
		mux.Handle("/count", httptransport.NewServer(
			countEndpoint,
			decodeCountRequest,
			encodeResponse,
			httptransport.ServerBefore(opentracing.FromHTTPRequest(tracer, "Count", logger)),
		))
	}

	return mux
}
