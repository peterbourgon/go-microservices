package main

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"
)

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

func decodeCombineRequest(_ context.Context, r *http.Request) (request interface{}, err error) {
	return combineRequest{A: r.FormValue("a"), B: r.FormValue("b")}, nil
}

func encodeCombineResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(combineResponse)
	if resp.Err != nil {
		http.Error(w, resp.Err.Error(), http.StatusInternalServerError)
		return nil
	}
	fmt.Fprintf(w, "%s\n", resp.V)
	return nil
}
