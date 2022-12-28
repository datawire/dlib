package dhttp_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

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
		if _, err := io.ReadAll(resp.Body); err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestShutdownIdle(t *testing.T) {
	httpScenarios(t, func(t *testing.T, url string, client *http.Client, server func(context.Context, *dhttp.ServerConfig) error) {
		ctx := dlog.NewTestContext(t, true)
		ctx, softCancel := context.WithCancel(dcontext.WithSoftness(ctx))

		sc := &dhttp.ServerConfig{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello world")
			}),
		}

		// Wrap the final Handler so that we can track when it cleans up.
		var workers sync.WaitGroup
		ctx = dhttp.WithTestHook(ctx, func(inner http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				workers.Add(1)
				defer workers.Done()
				inner.ServeHTTP(w, r)
			})
		})

		// Run the server.
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

		// Make a request in order to initiate the ServeConn.
		resp, err := client.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.ReadAll(resp.Body); err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}

		// Now shut down the server
		softCancel()
		// At this point, the workers should have been all cleaned up, so workers.Wait()
		// should return.  This will hang ServeConn isn't getting shut down correctly.
		workers.Wait()
	})
}

func TestShutdownActive(t *testing.T) {
	httpScenarios(t, func(t *testing.T, url string, client *http.Client, server func(context.Context, *dhttp.ServerConfig) error) {
		ctx, hardCancel := context.WithCancel(dlog.NewTestContext(t, true))
		defer hardCancel()
		ctx, softCancel := context.WithCancel(dcontext.WithSoftness(ctx))

		sRequestReceived := make(chan struct{})
		cRequestFinished := make(chan struct{})

		sc := &dhttp.ServerConfig{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(sRequestReceived)
				t.Log("debug: Received request")
				<-r.Context().Done()
				t.Log("debug: Canceled request")
				<-cRequestFinished
			}),
		}

		// Wrap the final Handler so that we can track when it cleans up.
		var workers sync.WaitGroup
		ctx = dhttp.WithTestHook(ctx, func(inner http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				workers.Add(1)
				defer workers.Done()
				inner.ServeHTTP(w, r)
			})
		})

		// Run the server.
		serverCh := make(chan error)
		go func() {
			serverCh <- server(ctx, sc)
		}()
		defer func() {
			if err := <-serverCh; err == nil {
				t.Error("error: expected an error from the server")
			}
		}()

		// Launch a request.  It should hang until the server is hard shutdown.
		workers.Add(1)
		go func() {
			defer workers.Done()
			resp, err := client.Get(url)
			if err == nil {
				t.Error("error: client expected an error")
			}
			if resp != nil {
				t.Errorf("error: client expected no response, but got: %v", resp)
			}
			close(cRequestFinished)
		}()

		<-sRequestReceived

		t.Log("debug: Shutdown...")
		softCancel()
		// Because the request hangs until the r.Context() is canceled, the Shutdown should
		// hang.  Let's check that it's hanging; 2s seems long enough to be sure.
		select {
		case <-serverCh:
			t.Fatal("error: server should be hanging, not returning!")
		case <-time.After(2 * time.Second):
		}

		// This should both cause the client to return with an error, and cause the
		// r.Context()s to be canceled, causing the handlers to return.
		t.Log("debug: Close...")
		hardCancel()
		workers.Wait()
	})
}
