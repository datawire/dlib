package dhttp_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

func httpScenarios(t *testing.T,
	testFn func(
		t *testing.T,
		url string,
		client *http.Client,
		server func(context.Context, *dhttp.ServerConfig) error,
	),
) {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	servers := map[string]struct {
		serve   func(net.Listener) func(context.Context, *dhttp.ServerConfig) error
		scheme  string
		clients map[string]func(net.Listener) http.RoundTripper
	}{
		"cleartext": {
			serve: func(ln net.Listener) func(ctx context.Context, sc *dhttp.ServerConfig) error {
				return func(ctx context.Context, sc *dhttp.ServerConfig) error {
					return sc.Serve(ctx, ln)
				}
			},
			scheme: "http",
			clients: map[string]func(net.Listener) http.RoundTripper{
				"h1": func(ln net.Listener) http.RoundTripper {
					ret := http.DefaultTransport.(*http.Transport).Clone()
					ret.DialContext = dialer.DialContext
					ret.DialTLSContext = func(_ context.Context, _, _ string) (net.Conn, error) {
						return nil, errors.New("should not be calling DialTLSContext for cleartext h1")
					}
					ret.ForceAttemptHTTP2 = false
					return ret
				},
			},
		},
		"tls": {
			serve: func(ln net.Listener) func(ctx context.Context, sc *dhttp.ServerConfig) error {
				return func(ctx context.Context, sc *dhttp.ServerConfig) error {
					certFile, keyFile, cleanup, err := testCertFiles()
					if err != nil {
						return err
					}
					defer cleanup()
					return sc.ServeTLS(ctx, ln, certFile, keyFile)
				}
			},
			scheme: "https",
			clients: map[string]func(net.Listener) http.RoundTripper{
				"h1": func(ln net.Listener) http.RoundTripper {
					ret := http.DefaultTransport.(*http.Transport).Clone()
					ret.TLSClientConfig = &tls.Config{
						InsecureSkipVerify: true,
					}
					ret.DialContext = dialer.DialContext
					ret.ForceAttemptHTTP2 = false
					return ret
				},
			},
		},
	}

	for sName, sDat := range servers {
		t.Run(sName, func(t *testing.T) {
			for cName, cDat := range sDat.clients {
				t.Run(cName, func(t *testing.T) {
					listener, err := net.Listen("tcp", "127.0.0.1:0")
					if err != nil {
						if listener, err = net.Listen("tcp6", "[::1]:0"); err != nil {
							t.Fatalf("httptest: failed to listen on a port: %v", err)
						}
					}

					u := sDat.scheme + "://" + listener.Addr().String()

					client := &http.Client{
						Transport: cDat(listener),
					}
					defer client.CloseIdleConnections()

					server := sDat.serve(listener)

					testFn(t, u, client, server)
				})
			}
		})
	}
}

// TestSmoketest is the most basic smoketest of different client/server scenarios.  Honestly, I
// wrote it more to test the httpScenarios test helper function itself, more than I wrote it to test
// dhttp.
func TestSmoketest(t *testing.T) {
	httpScenarios(t, func(t *testing.T, url string, client *http.Client, server func(context.Context, *dhttp.ServerConfig) error) {
		ctx, hardCancel := context.WithCancel(dlog.NewTestContext(t, true))
		defer hardCancel()
		ctx, softCancel := context.WithCancel(dcontext.WithSoftness(ctx))
		defer softCancel()

		sc := &dhttp.ServerConfig{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, "hello world")
			}),
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			assert.NoError(t, server(ctx, sc))
		}()
		go func() {
			defer wg.Done()
			defer softCancel()

			resp, err := client.Get(url)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.NotNil(t, resp) {
				return
			}
			if !assert.NotNil(t, resp.Body) {
				return
			}
			defer resp.Body.Close()

			wantedTLS := strings.HasPrefix(url, "https://")
			usedTLS := resp.TLS != nil
			assert.Equalf(t, wantedTLS, usedTLS, "wantedTLS=%v usedTLS=%v", wantedTLS, usedTLS)

			assert.Equal(t,
				strings.Contains(t.Name(), "h2"),
				resp.ProtoMajor == 2,
				"didn't get correct HTTP version")
		}()
		wg.Wait()
	})
}

// TestHardShutdown checks to make sure that the TCP connection gets forcefully closed when the
// server gets a hard-shutdown.
func TestHardShutdown(t *testing.T) {
	httpScenarios(t, func(t *testing.T, url string, client *http.Client, server func(context.Context, *dhttp.ServerConfig) error) {
		ctx, hardCancel := context.WithCancel(dlog.NewTestContext(t, false))
		defer hardCancel()
		ctx = dcontext.WithSoftness(ctx)

		sRequestReceived := make(chan struct{})
		sRequestCanceled := make(chan struct{})
		sRequestFinished := make(chan struct{})
		cRequestFinished := make(chan struct{})
		sExited := make(chan struct{})

		sc := &dhttp.ServerConfig{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(sRequestReceived)

				<-r.Context().Done()
				close(sRequestCanceled)

				// Waiting for <-cRequestFinished is important; we need to make sure
				// that a hard shutdown is triggerd when handlers hang.
				<-cRequestFinished
				close(sRequestFinished)
			}),
		}
		go func() {
			assert.Error(t, server(ctx, sc))
			close(sExited)
		}()
		go func() {
			resp, err := client.Get(url)
			// `err != nil` is important; if the request got interrupted then it's
			// important that it isn't just interpretted as a truncated response, and is
			// actually an error.
			assert.Error(t, err)
			assert.Nil(t, resp)
			close(cRequestFinished)
		}()

		<-sRequestReceived
		hardCancel()
		<-sRequestCanceled
		<-cRequestFinished
		<-sExited
		<-sRequestFinished
	})
}
