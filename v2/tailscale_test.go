// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tsclient_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

type TestServer struct {
	t *testing.T

	BaseURL *url.URL

	Method string
	Path   string
	Query  url.Values
	Body   *bytes.Buffer
	Header http.Header

	ResponseCode   int
	ResponseBody   interface{}
	ResponseHeader http.Header
}

func NewTestHarness(t *testing.T) (*tsclient.Client, *TestServer) {
	t.Helper()

	testServer := &TestServer{
		t:              t,
		ResponseHeader: make(http.Header),
	}

	mux := http.NewServeMux()
	mux.Handle("/", testServer)
	svr := &http.Server{
		Handler: mux,
	}

	// Start a listener on a random port
	listener, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)

	go func() {
		_ = svr.Serve(listener)
	}()

	// When the test is over, close the server
	t.Cleanup(func() {
		assert.NoError(t, svr.Close())
	})

	baseURL := fmt.Sprintf("http://localhost:%v", listener.Addr().(*net.TCPAddr).Port)
	testServer.BaseURL, err = url.Parse(baseURL)
	assert.NoError(t, err)
	client := &tsclient.Client{
		BaseURL: testServer.BaseURL,
		APIKey:  "not a real key",
		Tailnet: "example.com",
	}

	return client, testServer
}

func (t *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.Method = r.Method
	t.Path = r.URL.Path
	t.Query = r.URL.Query()
	t.Header = r.Header

	t.Body = bytes.NewBuffer([]byte{})
	_, err := io.Copy(t.Body, r.Body)
	assert.NoError(t.t, err)

	maps.Copy(w.Header(), t.ResponseHeader)
	w.WriteHeader(t.ResponseCode)
	if t.ResponseBody != nil {
		switch body := t.ResponseBody.(type) {
		case []byte:
			_, err := w.Write(body)
			assert.NoError(t.t, err)
		default:
			assert.NoError(t.t, json.NewEncoder(w).Encode(body))
		}
	}
}
