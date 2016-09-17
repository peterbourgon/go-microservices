package endpoints

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/tracing/opentracing"
	stdopentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"

	"github.com/peterbourgon/go-microservices/addsvc/pkg/service"
)

// New returns an Endpoints that wraps the provided server, and wires in all of
// the expected endpoint middlewares via the various parameters.
func New(svc service.Service, logger log.Logger, duration metrics.Histogram, trace stdopentracing.Tracer) Endpoints {
	var sumEndpoint endpoint.Endpoint
	{
		sumEndpoint = MakeSumEndpoint(svc)
		sumEndpoint = opentracing.TraceServer(trace, "Sum")(sumEndpoint)
		sumEndpoint = InstrumentingMiddleware(duration.With("method", "Sum"))(sumEndpoint)
		sumEndpoint = LoggingMiddleware(log.NewContext(logger).With("method", "Sum"))(sumEndpoint)
	}
	var concatEndpoint endpoint.Endpoint
	{
		concatEndpoint = MakeConcatEndpoint(svc)
		concatEndpoint = opentracing.TraceServer(trace, "Concat")(concatEndpoint)
		concatEndpoint = InstrumentingMiddleware(duration.With("method", "Concat"))(concatEndpoint)
		concatEndpoint = LoggingMiddleware(log.NewContext(logger).With("method", "Concat"))(concatEndpoint)
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

// Sum implements Service. Primarily useful in a client.
func (e Endpoints) Sum(ctx context.Context, a, b int) (int, error) {
	request := SumRequest{A: a, B: b}
	response, err := e.SumEndpoint(ctx, request)
	if err != nil {
		return 0, err
	}
	resp := response.(SumResponse)
	return resp.V, resp.Err
}

// Concat implements Service. Primarily useful in a client.
func (e Endpoints) Concat(ctx context.Context, a, b string) (string, error) {
	request := ConcatRequest{A: a, B: b}
	response, err := e.ConcatEndpoint(ctx, request)
	if err != nil {
		return "", err
	}
	resp := response.(ConcatResponse)
	return resp.V, resp.Err
}

// MakeSumEndpoint returns an endpoint that invokes Sum on the service.
// Primarily useful in a server.
func MakeSumEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		sumReq := request.(SumRequest)
		v, err := s.Sum(ctx, sumReq.A, sumReq.B)
		if err == service.ErrIntOverflow {
			return nil, err // special case; see comment on ErrIntOverflow
		}
		return SumResponse{V: v, Err: err}, nil
	}
}

// MakeConcatEndpoint returns an endpoint that invokes Concat on the service.
// Primarily useful in a server.
func MakeConcatEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		concatReq := request.(ConcatRequest)
		v, err := s.Concat(ctx, concatReq.A, concatReq.B)
		return ConcatResponse{V: v, Err: err}, nil
	}
}

// Failer is an interface that should be implemented by response types.
// Response encoders can check if responses are Failer, and if so if they've
// failed, and if so encode them using a separate write path based on the error.
type Failer interface {
	Failed() error
}

// SumRequest collects the request parameters for the Sum method.
type SumRequest struct {
	A, B int
}

// SumResponse collects the response values for the Sum method.
type SumResponse struct {
	V   int   `json:"v"`
	Err error `json:"-"` // should be intercepted by errorEncoder
}

// Failed implements Failer.
func (r SumResponse) Failed() error { return r.Err }

// ConcatRequest collects the request parameters for the Concat method.
type ConcatRequest struct {
	A, B string
}

// ConcatResponse collects the response values for the Concat method.
type ConcatResponse struct {
	V   string `json:"v"`
	Err error  `json:"-"` // should be intercepted by errorEncoder
}

// Failed implements Failer.
func (r ConcatResponse) Failed() error { return r.Err }
