package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/peterbourgon/go-microservices/addsvc/pkg/endpoints"
	addhttp "github.com/peterbourgon/go-microservices/addsvc/pkg/http"
	"github.com/peterbourgon/go-microservices/addsvc/pkg/service"
)

func main() {
	var (
		debugAddr     = flag.String("debug.addr", ":8080", "Debug and metrics listen address")
		httpAddr      = flag.String("http.addr", ":8081", "HTTP listen address")
		stringsvcAddr = flag.String("stringsvc.addr", "", "Optional host:port of a stringsvc")
		//tracerAddr    = flag.String("tracer.addr", "", "Enable Tracer tracing via a Tracer server host:port")
	)
	flag.Parse()

	// Logging domain.
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stdout)
		logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
		logger = log.NewContext(logger).With("caller", log.DefaultCaller)
	}
	logger.Log("msg", "hello")
	defer logger.Log("msg", "goodbye")

	// Metrics domain.
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
		//	trace = tracer.NewTracer("addsvc", storer, tracer.RandomID{})
		//} else {
		logger.Log("tracer", "none")
		trace = stdopentracing.GlobalTracer() // no-op
		//}
	}

	// Mechanical domain.
	errc := make(chan error)
	ctx := context.Background()

	// Business domain.
	svc := service.New(*stringsvcAddr, logger, ints, chars)

	// Endpoint domain.
	eps := endpoints.New(svc, logger, duration, trace)

	// Transport domain.
	mux := addhttp.NewHandler(ctx, eps, trace, logger)

	// Interrupt handler.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// Debug listener.
	go func() {
		m := http.NewServeMux()
		m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		m.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		m.Handle("/metrics", stdprometheus.Handler())

		logger.Log("transport", "debug", "addr", *debugAddr)
		errc <- http.ListenAndServe(*debugAddr, m)
	}()

	// HTTP transport.
	go func() {
		logger.Log("transport", "HTTP", "addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, mux)
	}()

	// Run!
	logger.Log("exit", <-errc)
}
