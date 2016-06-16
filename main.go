package main

import (
	"net/http"
	"os"
	"time"

	jujuratelimit "github.com/juju/ratelimit"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/ratelimit"
	httptransport "github.com/go-kit/kit/transport/http"
)

func main() {
	// Mechanical domain.
	ctx := context.Background()
	logger := log.NewLogfmtLogger(os.Stderr)
	errc := make(chan error)

	// Metrics domain.
	var requestCount metrics.Counter
	var requestDuration metrics.TimeHistogram
	{
		requestCount = prometheus.NewCounter(stdprometheus.CounterOpts{
			Namespace: "myteam",
			Subsystem: "myservice",
			Name:      "request_count",
			Help:      "Number of requests.",
		}, []string{"method", "success"})
		requestDuration = metrics.NewTimeHistogram(time.Nanosecond, prometheus.NewSummary(stdprometheus.SummaryOpts{
			Namespace: "myteam",
			Subsystem: "myservice",
			Name:      "request_duration_ns",
			Help:      "Request duration in nanoseconds.",
		}, []string{"method", "success"}))
	}

	// Service, or business-logic, domain.
	var s Service
	{
		s = service{}
		s = logging(logger)(s)
		s = instrumenting(requestCount, requestDuration)(s)
	}

	// Endpoint domain.
	var uppercase, count endpoint.Endpoint
	{
		uppercase = makeUppercaseEndpoint(s)
		uppercase = ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(5, 5))(uppercase)
	}
	{
		count = makeCountEndpoint(s)
		count = ratelimit.NewTokenBucketLimiter(jujuratelimit.NewBucketWithRate(3, 3))(count)
	}

	// Transport domain (HTTP).
	go func() {
		http.Handle("/uppercase", httptransport.NewServer(
			ctx,
			uppercase,
			decodeUppercaseRequest,
			encodeUppercaseResponse,
		))
		http.Handle("/count", httptransport.NewServer(
			ctx,
			count,
			decodeCountRequest,
			encodeCountResponse,
		))
		http.Handle("/metrics", stdprometheus.Handler())
		errc <- http.ListenAndServe(":8080", nil)
	}()

	logger.Log("err", <-errc)
}
