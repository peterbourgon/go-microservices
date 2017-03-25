package main

import (
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func main() {
	requests := []*http.Request{
		mustNewRequest("GET", "http://addsvc.default.svc.cluster.local/sum", strings.NewReader(`{"a":123,"b":456}`)),
		mustNewRequest("GET", "http://addsvc.default.svc.cluster.local/concat", strings.NewReader(`{"a":"hello","b":"world"}`)),
		mustNewRequest("GET", "http://stringsvc.default.svc.cluster.local/uppercase", strings.NewReader(`{"s":"blep"}`)),
		mustNewRequest("GET", "http://stringsvc.default.svc.cluster.local/count", strings.NewReader(`{"s":"🍔"}`)),
	}
	for begin := range time.Tick(time.Second) {
		req := requests[rand.Intn(len(requests))]
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Print(err)
			continue
		}
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := strings.TrimSpace(string(body))
		log.Printf("%s %s: %s (%s)", req.Method, req.URL.String(), bodyStr, time.Since(begin))
	}
}

func mustNewRequest(method, urlStr string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		panic(err)
	}
	return req
}
