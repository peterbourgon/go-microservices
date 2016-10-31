package service

import (
	"log"

	"github.com/go-kit/kit/metrics"
)

// Middleware describes a service (as opposed to endpoint) middleware.
type Middleware func(Service) Service

// LoggingMiddleware takes a logger as a dependency
// and returns a ServiceMiddleware.
func LoggingMiddleware(logger *log.Logger) Middleware {
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

func (mw instrumentingMiddleware) Sum(a, b int) (int, error) {
	v, err := mw.next.Sum(a, b)
	mw.ints.Add(float64(v))
	return v, err
}

func (mw instrumentingMiddleware) Concat(a, b string) (string, error) {
	v, err := mw.next.Concat(a, b)
	mw.chars.Add(float64(len(v)))
	return v, err
}
