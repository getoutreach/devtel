// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the Telefork client.

package telefork

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/getoutreach/gobox/pkg/trace"
)

// Client is the Telefork Service client.
type Client struct {
	http    *http.Client
	baseURL string
}

// NewClient returns a new Telefork client.
func NewClient(appName, apiKey string) *Client {
	return NewClientWithHTTPClient(appName, apiKey, http.DefaultClient)
}

// NewClientWithHTTPClient returns a new Telefork client with the given HTTP client.
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

// SendEvents sends the given events to Telefork.
func (c *Client) SendEvents(ctx context.Context, events []interface{}) error {
	ctx = trace.StartCall(ctx, "telefork.Client.SendEvents")
	defer trace.EndCall(ctx)

	b, err := json.Marshal(events)
	if err != nil {
		return trace.SetCallStatus(ctx, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(c.baseURL, "/")+"/", bytes.NewReader(b))
	if err != nil {
		return trace.SetCallStatus(ctx, err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return trace.SetCallStatus(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return trace.SetCallStatus(ctx, fmt.Errorf("Unexpected status code: %d", resp.StatusCode))
	}

	return trace.SetCallStatus(ctx, nil)
}

// Transport is an http.RoundTripper that adds the X-OUTREACH-CLIENT-APP-ID and X-OUTREACH-CLIENT-LOGGING headers.
type Transport struct {
	appName string
	apiKey  string

	original http.RoundTripper
}

// NewTransport returns a new Transport.
func NewTransport(appName, apiKey string, rt http.RoundTripper) *Transport {
	return &Transport{
		appName: appName,
		apiKey:  apiKey,

		original: rt,
	}
}

// RoundTrip implements http.RoundTripper.
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
