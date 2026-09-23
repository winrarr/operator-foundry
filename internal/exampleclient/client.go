/*
Copyright 2026 winrarr.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package exampleclient is a deliberately small typed client pattern.
package exampleclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotFound = errors.New("external resource not found")

type Resource struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Membership describes one relationship edge between external resources.
type Membership struct {
	ParentID string `json:"parentID"`
	MemberID string `json:"memberID"`
}

// MembershipKey returns a URL-safe key for the relationship edge.
func MembershipKey(parentID, memberID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(parentID + "\x00" + memberID))
}

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
}

func New(baseURL, token string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil {
		return nil, fmt.Errorf("parse external API endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("external API endpoint must use http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("external API endpoint has no host")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("external API token is empty")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL: parsed,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) Health(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/health", nil, nil)
}

func (c *Client) GetResource(ctx context.Context, name string) (*Resource, error) {
	var resource Resource
	if err := c.do(ctx, http.MethodGet, "/resources/"+url.PathEscape(name), nil, &resource); err != nil {
		return nil, err
	}
	return &resource, nil
}

func (c *Client) CreateResource(ctx context.Context, resource Resource) (*Resource, error) {
	var created Resource
	if err := c.do(ctx, http.MethodPost, "/resources", resource, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateResource(ctx context.Context, resource Resource) (*Resource, error) {
	var updated Resource
	if err := c.do(ctx, http.MethodPut, "/resources/"+url.PathEscape(resource.Name), resource, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteResource(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/resources/"+url.PathEscape(name), nil, nil)
}

// EnsureMembership creates or preserves exactly one relationship edge.
func (c *Client) EnsureMembership(ctx context.Context, parentID, memberID string) error {
	membership := Membership{ParentID: parentID, MemberID: memberID}
	return c.do(ctx, http.MethodPut, "/memberships/"+MembershipKey(parentID, memberID), membership, nil)
}

// DeleteMembership removes exactly one relationship edge.
func (c *Client) DeleteMembership(ctx context.Context, parentID, memberID string) error {
	return c.do(ctx, http.MethodDelete, "/memberships/"+MembershipKey(parentID, memberID), nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	requestURL := c.baseURL.ResolveReference(&url.URL{Path: strings.TrimPrefix(path, "/")})
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s request: %w", method, path, err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), requestBody)
	if err != nil {
		return fmt.Errorf("build %s %s request: %w", method, path, err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send %s %s request: %w", method, path, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("external API %s %s returned %s: %s", method, path, response.Status, strings.TrimSpace(string(message)))
	}
	if result == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}
