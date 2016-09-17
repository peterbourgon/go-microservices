package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"golang.org/x/net/context"
)

// Middleware describes a service (as opposed to endpoint) middleware.
type Middleware func(Service) Service

// NopMiddleware returns a no-op service middleware.
func NopMiddleware() Middleware {
	return func(next Service) Service { return next }
}

// LoggingMiddleware returns a service middleware that logs the
// parameters and result of each method invocation.
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{
			logger: logger,
			next:   next,
		}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) Sum(ctx context.Context, a, b int) (v int, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "Sum",
			"a", a, "b", b, "result", v, "error", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.Sum(ctx, a, b)
}

func (mw loggingMiddleware) Concat(ctx context.Context, a, b string) (v string, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "Concat",
			"a", a, "b", b, "result", v, "error", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.Concat(ctx, a, b)
}

// InstrumentingMiddleware returns a service middleware that instruments
// the number of integers summed and characters concatenated over the lifetime of
// the service.
func InstrumentingMiddleware(ints, chars metrics.Counter) Middleware {
	return func(next Service) Service {
		return instrumentingMiddleware{
			ints:  ints,
			chars: chars,
			next:  next,
		}
	}
}

type instrumentingMiddleware struct {
	ints  metrics.Counter
	chars metrics.Counter
	next  Service
}

func (mw instrumentingMiddleware) Sum(ctx context.Context, a, b int) (int, error) {
	v, err := mw.next.Sum(ctx, a, b)
	mw.ints.Add(float64(v))
	return v, err
}

func (mw instrumentingMiddleware) Concat(ctx context.Context, a, b string) (string, error) {
	v, err := mw.next.Concat(ctx, a, b)
	mw.chars.Add(float64(len(v)))
	return v, err
}

// RemoteUppercasingMiddleware returns a middleware that uppercases the result
// of the concat method using a remote stringsvc to do the heavy lifting.
func RemoteUppercasingMiddleware(endpoint string) Middleware {
	return func(next Service) Service {
		return &uppercasingMiddleware{
			Service: next,
			uppercase: func(s string) (string, error) {
				if !strings.HasPrefix(endpoint, "http") {
					endpoint = "http://" + endpoint
				}
				u, err := url.Parse(endpoint)
				if err != nil {
					return "", err
				}
				var buf bytes.Buffer
				if err := json.NewEncoder(&buf).Encode(map[string]string{
					"s": s,
				}); err != nil {
					return "", err
				}
				u.Path = "uppercase"
				req, err := http.NewRequest("GET", u.String(), &buf)
				if err != nil {
					return "", err
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return "", err
				}
				defer resp.Body.Close()
				var response struct {
					V string `json:"v"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
					return "", err
				}
				return response.V, nil
			},
		}
	}
}

type uppercasingMiddleware struct {
	Service
	uppercase func(string) (string, error)
}

func (mw uppercasingMiddleware) Concat(ctx context.Context, a, b string) (string, error) {
	v, err := mw.Service.Concat(ctx, a, b)
	if err != nil {
		return v, err
	}
	return mw.uppercase(v)
}
