package service

import "log"

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
