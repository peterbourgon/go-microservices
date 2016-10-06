package endpoints

import (
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/ratelimit"
	rl "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"

	"github.com/peterbourgon/go-microservices/pkg/service"
)

// New returns an Endpoints that wraps the provided server, and wires in all of
// the expected endpoint middlewares via the various parameters.
func New(svc service.Service, logger log.Logger, duration metrics.Histogram) Endpoints {
	var sumEndpoint endpoint.Endpoint
	{
		sumEndpoint = MakeSumEndpoint(svc)
		sumEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(1, 1))(sumEndpoint)
		sumEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(sumEndpoint)
		sumEndpoint = LoggingMiddleware(log.NewContext(logger).With("method", "Sum"))(sumEndpoint)
		sumEndpoint = InstrumentingMiddleware(duration.With("method", "Sum"))(sumEndpoint)
	}
	var concatEndpoint endpoint.Endpoint
	{
		concatEndpoint = MakeConcatEndpoint(svc)
		concatEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(100, 100))(concatEndpoint)
		concatEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(concatEndpoint)
		concatEndpoint = LoggingMiddleware(log.NewContext(logger).With("method", "Concat"))(concatEndpoint)
		concatEndpoint = InstrumentingMiddleware(duration.With("method", "Concat"))(concatEndpoint)
	}
	return Endpoints{
		SumEndpoint:    sumEndpoint,
		ConcatEndpoint: concatEndpoint,
	}
}

// Endpoints collects all of the endpoints that compose an add service. It's
// meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type Endpoints struct {
	SumEndpoint    endpoint.Endpoint
	ConcatEndpoint endpoint.Endpoint
}

// MakeSumEndpoint constructs a Sum endpoint wrapping the service.
func MakeSumEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(SumRequest)
		v, err := s.Sum(ctx, req.A, req.B)
		return SumResponse{V: v, Err: err}, nil
	}
}

// MakeConcatEndpoint constructs a Concat endpoint wrapping the service.
func MakeConcatEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(ConcatRequest)
		v, err := s.Concat(ctx, req.A, req.B)
		return ConcatResponse{V: v, Err: err}, nil
	}
}

// SumRequest collects the request parameters for the Sum method.
type SumRequest struct {
	A, B int
}

// SumResponse collects the response values for the Sum method.
type SumResponse struct {
	V   int   `json:"v"`
	Err error `json:"err"`
}

// ConcatRequest collects the request parameters for the Concat method.
type ConcatRequest struct {
	A, B string
}

// ConcatResponse collects the response values for the Concat method.
type ConcatResponse struct {
	V   string `json:"v"`
	Err error  `json:"err"`
}
