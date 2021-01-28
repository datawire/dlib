package dhttp_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

func TestContext(t *testing.T) {
	httpScenarios(t, func(t *testing.T, url string, client *http.Client, server func(context.Context, *dhttp.ServerConfig) error) {
		type testContextKey struct{}
		ctx := context.WithValue(dlog.NewTestContext(t, true), testContextKey{}, "testvalue")
		ctx, softCancel := context.WithCancel(dcontext.WithSoftness(ctx))

		sc := &dhttp.ServerConfig{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Context().Value(testContextKey{}) != "testvalue" {
					t.Errorf("Request doesn't have expected base context: %v", r.Context())
				}
				fmt.Fprint(w, "Hello world")
			}),
		}

		serverCh := make(chan error)
		go func() {
			serverCh <- server(ctx, sc)
		}()
		defer func() {
			softCancel()
			if err := <-serverCh; err != nil {
				t.Error(err)
			}
		}()

		resp, err := client.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	})
}
