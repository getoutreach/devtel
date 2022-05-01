package telefork

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient(apiKey, appName string) *Client {
	return NewClientWithHTTPClient(appName, apiKey, http.DefaultClient)
}

func NewClientWithHTTPClient(appName, apiKey string, client *http.Client) *Client {
	baseURL := "https://telefork.outreach.io/"
	if os.Getenv("OUTREACH_TELEFORK_ENDPOINT") != "" {
		baseURL = os.Getenv("OUTREACH_TELEFORK_ENDPOINT")
	}

	client.Transport = NewTransport(appName, apiKey, client.Transport)
	return &Client{
		http:    client,
		baseURL: baseURL,
	}
}

func (c *Client) SendEvents(events []interface{}) error {
	b, err := json.Marshal(events)
	if err != nil {
		return err
	}

	resp, err := c.http.Post(strings.TrimSuffix(c.baseURL, "/")+"/", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

type Transport struct {
	appName string
	apiKey  string

	original http.RoundTripper
}

func NewTransport(appName, apiKey string, rt http.RoundTripper) *Transport {
	return &Transport{
		appName: appName,
		apiKey:  apiKey,

		original: rt,
	}
}

func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt := t.original
	if rt == nil {
		rt = http.DefaultTransport
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-OUTREACH-CLIENT-LOGGING", t.apiKey)
	r.Header.Set("X-OUTREACH-CLIENT-APP-ID", t.appName)

	return rt.RoundTrip(r)
}
