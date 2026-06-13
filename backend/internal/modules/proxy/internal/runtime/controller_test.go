package runtime

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestControllerClientVersionUsesBearerSecret(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "127.0.0.1:18001", req.URL.Host)
		require.Equal(t, "/version", req.URL.Path)
		require.Equal(t, "Bearer secret", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(`{"version":"v1.2.3"}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	info, err := (ControllerClient{HTTPClient: client}).Version(context.Background(), Instance{ControllerPort: 18001}, "secret")
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", info.Version)
}
