package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
)

type Service interface {
	Uppercase(string) (string, error)
	Count(string) int
}

type service struct{}

func (service) Uppercase(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty")
	}
	return strings.ToUpper(s), nil
}

func (service) Count(s string) int {
	return len(s)
}

type uppercaseRequest struct {
	S string
}

type uppercaseResponse struct {
	V   string
	Err error
}

func makeUppercaseEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(uppercaseRequest)
		v, err := s.Uppercase(req.S)
		return uppercaseResponse{V: v, Err: err}, nil
	}
}

type countRequest struct {
	S string
}

type countResponse struct {
	N int
}

func makeCountEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(countRequest)
		n := s.Count(req.S)
		return countResponse{N: n}, nil
	}
}

func main() {
	ctx := context.Background()
	logger := log.NewLogfmtLogger(os.Stderr)

	var s Service
	{
		s = service{}
		s = logging(logger)(s)
	}

	http.Handle("/uppercase", httptransport.NewServer(
		ctx,
		makeUppercaseEndpoint(s),
		decodeUppercaseRequest,
		encodeUppercaseResponse,
	))
	http.Handle("/count", httptransport.NewServer(
		ctx,
		makeCountEndpoint(s),
		decodeCountRequest,
		encodeCountResponse,
	))

	logger.Log("transport", "HTTP", "addr", ":8080")
	logger.Log("err", http.ListenAndServe(":8080", nil))
}

func decodeUppercaseRequest(_ context.Context, r *http.Request) (request interface{}, err error) {
	return uppercaseRequest{S: r.FormValue("s")}, nil
}

func encodeUppercaseResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(uppercaseResponse)
	if resp.Err != nil {
		http.Error(w, resp.Err.Error(), http.StatusBadRequest)
		return nil
	}
	fmt.Fprintf(w, "%s\n", resp.V)
	return nil
}

func decodeCountRequest(_ context.Context, r *http.Request) (request interface{}, err error) {
	return countRequest{S: r.FormValue("s")}, nil
}

func encodeCountResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(countResponse)
	fmt.Fprintf(w, "%d\n", resp.N)
	return nil
}

//
//
//

type Middleware func(Service) Service

func logging(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{next: next, logger: logger}
	}
}

type loggingMiddleware struct {
	next   Service
	logger log.Logger
}

func (mw loggingMiddleware) Uppercase(s string) (v string, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "Uppercase",
			"s", s,
			"v", v,
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.Uppercase(s)
}

func (mw loggingMiddleware) Count(s string) int {
	begin := time.Now()

	i := mw.next.Count(s)

	mw.logger.Log("method", "Count", "s", s, "i", i, "took", time.Since(begin))
	return i
}
