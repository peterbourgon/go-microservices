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
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

func main() {
	// Configuration from the environment.
	var (
		httpAddr = flag.String("http.addr", ":8081", "HTTP listen address")
		//tracerAddr = flag.String("tracer.addr", "", "Enable Tracer tracing via a Tracer server host:port")
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

	// Tracing domain.
	var trace stdopentracing.Tracer
	{
		//if *tracerAddr != "" {
		//	logger.Log("tracer", *tracerAddr)
		//	storer, err := tracer.NewGRPC(*tracerAddr, &tracer.GRPCOptions{
		//		QueueSize:     1024,
		//		FlushInterval: time.Second,
		//	}, grpc.WithInsecure())
		//	if err != nil {
		//		logger.Log("err", err)
		//		os.Exit(1)
		//	}
		//	trace = tracer.NewTracer("stringsvc", storer, tracer.RandomID{})
		//} else {
		logger.Log("tracer", "none")
		trace = stdopentracing.GlobalTracer() // no-op
		//}
	}

	// Construct the service.
	mux := makeServeMux(logger, requestCount, requestLatency, countResult, trace)
	mux.Handle("/metrics", stdprometheus.Handler())

	// Go!
	logger.Log("transport", "HTTP", "addr", *httpAddr)
	logger.Log("exit", http.ListenAndServe(*httpAddr, mux))
}

func makeServeMux(
	logger log.Logger,
	requestCount metrics.Counter,
	requestLatency, countResult metrics.Histogram,
	trace stdopentracing.Tracer,
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
		uppercaseEndpoint = opentracing.TraceServer(trace, "Uppercase")(uppercaseEndpoint)
	}
	var countEndpoint endpoint.Endpoint
	{
		countEndpoint = makeCountEndpoint(svc)
		countEndpoint = opentracing.TraceServer(trace, "Count")(countEndpoint)
	}

	// Transport domain.
	mux := http.NewServeMux()
	{
		uppercaseHandler := httptransport.NewServer(
			uppercaseEndpoint,
			decodeUppercaseRequest,
			encodeResponse,
		)
		countHandler := httptransport.NewServer(
			countEndpoint,
			decodeCountRequest,
			encodeResponse,
		)
		mux.Handle("/uppercase", uppercaseHandler)
		mux.Handle("/count", countHandler)
	}

	return mux
}
