package botengine

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/xsyetopz/imotherbtw/internal/integrations/kawaii"
)

type KawaiiConfig struct {
	Token string
}

type kawaiiClient struct {
	c *kawaii.Client
}

const kawaiiHTTPTimeout = 3 * time.Second

func newKawaiiClient(cfg KawaiiConfig) (*kawaiiClient, error) {
	c, err := kawaii.New(kawaii.Options{
		Token: cfg.Token,
		// BaseURL is fixed; host allowlist enforced by the client.
		BaseURL: "https://kawaii.red",
		HTTPClient: &http.Client{
			Timeout: kawaiiHTTPTimeout,
		},
	})
	if err != nil {
		return nil, err
	}
	return &kawaiiClient{c: c}, nil
}

func (k *kawaiiClient) FetchGIF(ctx context.Context, endpoint kawaii.Endpoint) (string, error) {
	if k == nil || k.c == nil {
		return "", errors.New("kawaii client not configured")
	}
	return k.c.FetchGIF(ctx, endpoint)
}
