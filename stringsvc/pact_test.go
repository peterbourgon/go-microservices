package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/discard"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/pact-foundation/pact-go/types"
	"github.com/pact-foundation/pact-go/utils"
)

func TestConsumers(t *testing.T) {
	pactFiles := getPactFiles(t)
	if len(pactFiles) == 0 {
		t.Skip("no Pact files found")
	}

	mux := makeServeMux(
		log.NewNopLogger(),
		discard.NewCounter(),
		discard.NewHistogram(),
		discard.NewHistogram(),
		opentracing.GlobalTracer(),
	)
	mux.HandleFunc("/setup", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	})
	mux.HandleFunc("/states", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, "{}")
	})

	// Set up the WaitGroup, and put the wait on the defer stack.
	var serverWait sync.WaitGroup
	serverWait.Add(1)
	defer serverWait.Wait() // Order-of-operations: 3

	// Bind the listener, and put the close on the defer stack.
	// The close must pop off the defer stack before the wait!
	port, _ := utils.GetFreePort()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close() // Order-of-operations: 1

	go func() {
		// Signal the waitgroup to finish when the HTTP server closes.
		defer serverWait.Done() // Order-of-operations: 2
		t.Logf("API starting: port %d (%s)", port, ln.Addr())
		t.Logf("API terminating: %v", http.Serve(ln, mux))
	}()

	pact := dsl.Pact{
		Port: 6666,
	}
	if err := pact.VerifyProvider(types.VerifyRequest{
		ProviderBaseURL:        fmt.Sprintf("http://localhost:%d", port),
		PactURLs:               pactFiles,
		ProviderStatesURL:      fmt.Sprintf("http://localhost:%d/states", port),
		ProviderStatesSetupURL: fmt.Sprintf("http://localhost:%d/setup", port),
	}); err != nil {
		t.Errorf("verification failed: %v", err)
	}
}

func getPactFiles(t *testing.T) (res []string) {
	filepath.Walk(".", func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			return nil
		}
		if strings.Contains(path, "pact") && filepath.Ext(path) == ".json" {
			t.Logf("verifying Pact %s", path)
			abspath, _ := filepath.Abs(path)
			res = append(res, abspath)
		}
		return nil
	})
	return res
}
