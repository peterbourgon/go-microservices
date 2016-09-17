package service

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

func TestStringsvc(t *testing.T) {
	pact := dsl.Pact{
		Port:     6666,
		Consumer: "addsvc",
		Provider: "stringsvc",
	}
	defer pact.Teardown()

	pact.AddInteraction().
		UponReceiving("stringsvc uppercase").
		WithRequest(dsl.Request{
			Method: "post",
			Path:   "/uppercase",
			Body:   `{"s":"foo"}`,
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
			Body:   `{"v":"FOO"}`,
		})

	if err := pact.Verify(func() error {
		u := fmt.Sprintf("http://localhost:%d/uppercase", pact.Server.Port)
		req, err := http.NewRequest("GET", u, strings.NewReader(`{"s":"foo"}`))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		if _, err = http.DefaultClient.Do(req); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	pact.WritePact()
}
