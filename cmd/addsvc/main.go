package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"

	"github.com/peterbourgon/go-microservices/pkg/endpoints"
	addhttp "github.com/peterbourgon/go-microservices/pkg/http"
	"github.com/peterbourgon/go-microservices/pkg/service"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	svc := service.New(log.New(os.Stderr, "", log.LstdFlags))
	eps := endpoints.New(svc)
	mux := addhttp.NewHandler(context.Background(), eps)

	// There's no magic here. Just construction, passing deps as params.
	// Everything is explcit. Hopefully a breath of fresh air!

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
