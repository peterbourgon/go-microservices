package main

import (
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
)

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

type combineRequest struct {
	A, B string
}

type combineResponse struct {
	V   string
	Err error
}

func makeCombineEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(combineRequest)
		v, err := s.Combine(req.A, req.B)
		return combineResponse{V: v, Err: err}, nil
	}
}
