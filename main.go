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

func (s basicService) Sum(a, b int) (v int, err error) {
	defer func() {
		log.Printf("Sum(%d, %d) = %d, %v", a, b, v, err)
	}()
	return a + b, nil
}

func (s basicService) Concat(a, b string) (v string, err error) {
	defer func() {
		log.Printf("Concat(%q, %q) = %q, %v", a, b, v, err)
	}()
	return a + b, nil
}

// We get endpoints by writing endpoit constructors.
// This keeps the service pure, and avoids mixing in
// endpoint concerns.

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

func demonstrateEndpointMiddlewares() {
	// Now that we have endpoints, we can have endpoint middlewares!
	var _ endpoint.Middleware

	// Here's how you'd use them.
	var e endpoint.Endpoint
	e = makeSumEndpoint(basicService{})
	e = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(100, 100))(e) // Dive to definition!
	e = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(e)
}

// We can also define service middlewares,
// custom to our application.

type ServiceMiddleware func(Service) Service

// LoggingMiddleware takes a logger as a dependency
// and returns a ServiceMiddleware.
func LoggingMiddleware(logger *log.Logger) ServiceMiddleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

// loggingMiddleware implements a logging service middleware.
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

// Now we can delete the logging in the
// basicService implementation! Yay!

func demonstrateServiceMiddlewares() {
	var s Service
	s = basicService{}
	s = LoggingMiddleware(log.New(os.Stderr, "", log.LstdFlags))(s)
}

// We've built all this structure,
// but it's not yet exposed.

func (s basicService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sum":
		var req struct {
			A int `json:"a"`
			B int `json:"b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			code := http.StatusBadRequest
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		v, err := s.Sum(req.A, req.B)
		if err != nil {
			code := http.StatusInternalServerError
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]int{"v": v})
		log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, 200)

	case "/concat":
		var req struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			code := http.StatusBadRequest
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		v, err := s.Concat(req.A, req.B)
		if err != nil {
			code := http.StatusInternalServerError
			log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, code)
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]string{"v": v})
		log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, 200)

	default:
		log.Printf("%s: %s %s: %d", r.RemoteAddr, r.Method, r.URL, 400)
		http.NotFound(w, r)
	}
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, basicService{}))
}
