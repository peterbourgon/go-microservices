package http

// This file provides server-side bindings for the HTTP transport.
// It utilizes the transport/http.Server.

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
	httptransport "github.com/go-kit/kit/transport/http"
	stdopentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"

	"github.com/peterbourgon/go-microservices/addsvc/pkg/endpoints"
	"github.com/peterbourgon/go-microservices/addsvc/pkg/service"
)

// NewHandler returns a handler that makes a set of endpoints available on
// predefined paths.
func NewHandler(ctx context.Context, endpoints endpoints.Endpoints, tracer stdopentracing.Tracer, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorLogger(logger),
	}
	m := http.NewServeMux()
	m.Handle("/sum", httptransport.NewServer(
		ctx,
		endpoints.SumEndpoint,
		DecodeHTTPSumRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.FromHTTPRequest(tracer, "Sum", logger)))...,
	))
	m.Handle("/concat", httptransport.NewServer(
		ctx,
		endpoints.ConcatEndpoint,
		DecodeHTTPConcatRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.FromHTTPRequest(tracer, "Concat", logger)))...,
	))
	return m
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func err2code(err error) int {
	switch err {
	case service.ErrTwoZeroes, service.ErrMaxSizeExceeded, service.ErrIntOverflow:
		return http.StatusBadRequest
	}
	switch e := err.(type) {
	case httptransport.Error:
		switch e.Domain {
		case httptransport.DomainDecode:
			return http.StatusBadRequest
		case httptransport.DomainDo:
			return err2code(e.Err)
		}
	}
	return http.StatusInternalServerError
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string `json:"error"`
}

// DecodeHTTPSumRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded sum request from the HTTP request body. Primarily useful in a
// server.
func DecodeHTTPSumRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoints.SumRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// DecodeHTTPConcatRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded concat request from the HTTP request body. Primarily useful in a
// server.
func DecodeHTTPConcatRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoints.ConcatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// DecodeHTTPSumResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded sum response from the HTTP response body. If the response has a
// non-200 status code, we will interpret that as an error and attempt to decode
// the specific error message from the response body. Primarily useful in a
// client.
func DecodeHTTPSumResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp endpoints.SumResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// DecodeHTTPConcatResponse is a transport/http.DecodeResponseFunc that decodes
// a JSON-encoded concat response from the HTTP response body. If the response
// has a non-200 status code, we will interpret that as an error and attempt to
// decode the specific error message from the response body. Primarily useful in
// a client.
func DecodeHTTPConcatResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp endpoints.ConcatResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// EncodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that
// JSON-encodes any request to the request body. Primarily useful in a client.
func EncodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// EncodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func EncodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(endpoints.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), w)
		return nil
	}
	return json.NewEncoder(w).Encode(response)
}
