package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/peterbourgon/go-microservices/pkg/endpoints"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
)

// NewHandler returns a handler that makes a set of endpoints available on
// predefined paths.
func NewHandler(ctx context.Context, endpoints endpoints.Endpoints) http.Handler {
	m := http.NewServeMux()
	m.Handle("/sum", httptransport.NewServer(
		ctx,
		endpoints.SumEndpoint,
		DecodeSumRequest,
		EncodeGenericResponse,
	))
	m.Handle("/concat", httptransport.NewServer(
		ctx,
		endpoints.ConcatEndpoint,
		DecodeConcatRequest,
		EncodeGenericResponse,
	))
	m.Handle("/metrics", promhttp.Handler())
	return m
}

// DecodeSumRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded sum request from the HTTP request body. Primarily useful in a
// server.
func DecodeSumRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoints.SumRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// DecodeConcatRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded concat request from the HTTP request body. Primarily useful in a
// server.
func DecodeConcatRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoints.ConcatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// DecodeSumResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded sum response from the HTTP response body. If the response has a
// non-200 status code, we will interpret that as an error and attempt to decode
// the specific error message from the response body. Primarily useful in a
// client.
func DecodeSumResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}
	var resp endpoints.SumResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// DecodeConcatResponse is a transport/http.DecodeResponseFunc that decodes
// a JSON-encoded concat response from the HTTP response body. If the response
// has a non-200 status code, we will interpret that as an error and attempt to
// decode the specific error message from the response body. Primarily useful in
// a client.
func DecodeConcatResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}
	var resp endpoints.ConcatResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// EncodeGenericRequest is a transport/http.EncodeRequestFunc that
// JSON-encodes any request to the request body. Primarily useful in a client.
func EncodeGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// EncodeGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func EncodeGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
