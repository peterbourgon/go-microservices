package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/ratelimit"
	httptransport "github.com/go-kit/kit/transport/http"
	rl "github.com/juju/ratelimit"
	"github.com/sony/gobreaker"
	"golang.org/x/net/context"
)

// Service describes a service that adds things together.
type Service interface {
	Sum(a, b int) (int, error)
	Concat(a, b string) (string, error)
}

// basicService implements Service.
type basicService struct{}

func (s basicService) Sum(a, b int) (v int, err error) { return a + b, nil }

func (s basicService) Concat(a, b string) (v string, err error) { return a + b, nil }

func makeSumEndpoint(s Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (response interface{}, err error) {
		req := request.(SumRequest)
		v, err := s.Sum(req.A, req.B)
		return SumResponse{V: v, Err: err}, nil
	}
}

func makeConcatEndpoint(s Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (response interface{}, err error) {
		req := request.(ConcatRequest)
		v, err := s.Concat(req.A, req.B)
		return ConcatResponse{V: v, Err: err}, nil
	}
}

type SumRequest struct {
	A, B int
}

type SumResponse struct {
	V   int   `json:"v"`
	Err error `json:"err"`
}

type ConcatRequest struct {
	A, B string
}

type ConcatResponse struct {
	V   string `json:"v"`
	Err error  `json:"err"`
}

type ServiceMiddleware func(Service) Service

// LoggingMiddleware takes a logger as a dependency
// and returns a ServiceMiddleware.
func LoggingMiddleware(logger *log.Logger) ServiceMiddleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger *log.Logger
	next   Service
}

func (mw loggingMiddleware) Sum(a, b int) (v int, err error) {
	defer func() {
		mw.logger.Printf("Sum(%d, %d) = %d, %v", a, b, v, err) // single purpose
	}()
	return mw.next.Sum(a, b)
}

func (mw loggingMiddleware) Concat(a, b string) (v string, err error) {
	defer func() {
		mw.logger.Printf("Concat(%q, %q) = %q, %v", a, b, v, err)
	}()
	return mw.next.Concat(a, b)
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	// This is another Go kit idiom: wiring everything up in a big func main.
	// Start at the middle of the onion: the business logic, the service.
	var s Service
	{
		s = basicService{}
		s = LoggingMiddleware(log.New(os.Stderr, "", log.LstdFlags))(s)
	}

	// Then, create our two endpoints, wrapping the service.
	var sumEndpoint endpoint.Endpoint
	{
		sumEndpoint = makeSumEndpoint(s)
		sumEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(1, 1))(sumEndpoint)
		sumEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(sumEndpoint)
	}
	var concatEndpoint endpoint.Endpoint
	{
		concatEndpoint = makeConcatEndpoint(s)
		concatEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(100, 100))(concatEndpoint)
		concatEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(concatEndpoint)
	}

	// Now that we have endpoints, we can leverage Go kit's package transport.
	// Let's use the HTTP transport.
	mux := http.NewServeMux()
	{
		mux.Handle("/sum", httptransport.NewServer(
			context.Background(),
			sumEndpoint,
			decodeSumRequest,
			encodeSumResponse,
		))
		mux.Handle("/concat", httptransport.NewServer(
			context.Background(),
			concatEndpoint,
			decodeConcatRequest,
			encodeConcatResponse,
		))
	}

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// These functions are just extracted from our previous ServeHTTP method,
// leveraging the new types we've defined.

func decodeSumRequest(_ context.Context, r *http.Request) (request interface{}, err error) {
	var req SumRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func decodeConcatRequest(_ context.Context, r *http.Request) (request interface{}, err error) {
	var req ConcatRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func encodeSumResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func encodeConcatResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
