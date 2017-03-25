package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func main() {
	requests := []struct {
		method string
		url    string
		body   string
	}{
		{"GET", "http://addsvc.default.svc.cluster.local/sum", `{"a":123,"b":456}`},
		{"GET", "http://addsvc.default.svc.cluster.local/concat", `{"a":"hello","b":"world"}`},
		{"GET", "http://stringsvc.default.svc.cluster.local/uppercase", `{"s":"blep"}`},
		{"GET", "http://stringsvc.default.svc.cluster.local/count", `{"s":"🍔"}`},
	}
	for begin := range time.Tick(time.Second) {
		r := requests[rand.Intn(len(requests))]

		req, err := http.NewRequest(r.method, r.url, strings.NewReader(r.body))
		if err != nil {
			log.Printf("NewRequest: %v", err)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("DefaultClient.Do(%s %s): %v", r.method, r.url, err)
			continue
		}

		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := strings.TrimSpace(string(body))
		log.Printf(
			"%s %s: %s: %s (%s)",
			req.Method, req.URL.String(),
			resp.Status,
			bodyStr,
			time.Since(begin),
		)
	}
}
