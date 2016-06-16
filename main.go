package main

import (
	"errors"
	"fmt"
	"net/http"
)

type Service interface {
	Count(string) int
	Concat(a, b string) (string, error)
}

type service struct{}

func (service) Count(s string) int {
	return len(s)
}

func (service) Concat(a, b string) (string, error) {
	if a == "" && b == "" {
		return "", errors.New("two empty strings :(")
	}
	return a + b, nil
}

func main() {
	s := service{}
	http.HandleFunc("/count", count(s))
	http.HandleFunc("/concat", concat(s))
	http.ListenAndServe(":8080", nil)
}

func count(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		str := r.FormValue("s")
		count := s.Count(str)
		fmt.Fprintf(w, "%d\n", count)
	}
}

func concat(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, b := r.FormValue("a"), r.FormValue("b")
		v, err := s.Concat(a, b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "%s\n", v)
	}
}
