package kawaii_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/xsuetopz/go-mamusiabtw/internal/integrations/kawaii"
)

func TestFetchGIF(t *testing.T) {
	t.Parallel()

	c, err := kawaii.New(kawaii.Options{
		Token:   "t",
		BaseURL: "https://kawaii.red",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/api/gif/hug" {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
				}, nil
			}
			if r.URL.Query().Get("token") != "t" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader("unauthorized")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"response":"https://cdn.example/test.gif"}`)),
			}, nil
		})},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := c.FetchGIF(context.Background(), kawaii.EndpointHug)
	if err != nil {
		t.Fatalf("FetchGIF: %v", err)
	}
	if got != "https://cdn.example/test.gif" {
		t.Fatalf("unexpected url: %q", got)
	}
}

func TestNewRejectsNonHTTPS(t *testing.T) {
	t.Parallel()
	if _, err := kawaii.New(kawaii.Options{BaseURL: "http://kawaii.red"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewRejectsHost(t *testing.T) {
	t.Parallel()
	if _, err := kawaii.New(kawaii.Options{BaseURL: "https://example.com"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFetchRejectsEndpoint(t *testing.T) {
	t.Parallel()
	c, err := kawaii.New(kawaii.Options{Token: "anonymous"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, fetchErr := c.FetchGIF(context.Background(), kawaii.Endpoint("nope")); fetchErr == nil {
		t.Fatalf("expected error")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
