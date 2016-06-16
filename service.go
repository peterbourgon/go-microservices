package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
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

func (mw loggingMiddleware) Uppercase(s string) (string, error) {
	begin := time.Now()
	v, err := mw.next.Uppercase(s)
	mw.logger.Log("method", "Uppercase", "s", s, "v", v, "err", err, "took", time.Since(begin))
	return v, err
}

func (mw loggingMiddleware) Count(s string) int {
	begin := time.Now()
	i := mw.next.Count(s)
	mw.logger.Log("method", "Count", "s", s, "i", i, "took", time.Since(begin))
	return i
}

func instrumenting(count metrics.Counter, duration metrics.TimeHistogram) Middleware {
	return func(next Service) Service {
		return instrumentingMiddleware{next: next, count: count, duration: duration}
	}
}

type instrumentingMiddleware struct {
	next     Service
	count    metrics.Counter
	duration metrics.TimeHistogram
}

func (mw instrumentingMiddleware) Uppercase(s string) (v string, err error) {
	defer func(begin time.Time) {
		method := metrics.Field{Key: "method", Value: "Uppercase"}
		success := metrics.Field{Key: "success", Value: fmt.Sprint(err == nil)}
		mw.count.With(method).With(success).Add(1)
		mw.duration.With(method).With(success).Observe(time.Since(begin))
	}(time.Now())

	return mw.next.Uppercase(s)
}

func (mw instrumentingMiddleware) Count(s string) int {
	defer func(begin time.Time) {
		method := metrics.Field{Key: "method", Value: "Count"}
		success := metrics.Field{Key: "success", Value: "true"}
		mw.count.With(method).With(success).Add(1)
		mw.duration.With(method).With(success).Observe(time.Since(begin))
	}(time.Now())

	return mw.next.Count(s)
}
