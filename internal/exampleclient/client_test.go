package exampleclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientUsesTypedRequestsAndBearerAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/resources/widget" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"resource-1","name":"widget","value":"desired"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	resource, err := client.GetResource(context.Background(), "widget")
	if err != nil {
		t.Fatalf("get resource: %v", err)
	}
	if resource.ID != "resource-1" || resource.Value != "desired" {
		t.Fatalf("unexpected resource: %#v", resource)
	}
}

func TestClientClassifiesNotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := New(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	_, err = client.GetResource(context.Background(), "missing")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClientSupportsHealthCreateUpdateAndDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/health":
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodPost && request.URL.Path == "/resources":
			var resource Resource
			if err := json.NewDecoder(request.Body).Decode(&resource); err != nil {
				t.Errorf("decode create request: %v", err)
			}
			resource.ID = "created-1"
			writeTestJSON(writer, http.StatusCreated, resource)
		case request.Method == http.MethodPut && request.URL.Path == "/resources/widget":
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Errorf("read update request: %v", err)
			}
			if !strings.Contains(string(body), `"value":"updated"`) {
				t.Errorf("update request did not contain desired value: %s", body)
			}
			writeTestJSON(writer, http.StatusOK, Resource{ID: "created-1", Name: "widget", Value: "updated"})
		case request.Method == http.MethodDelete && request.URL.Path == "/resources/widget":
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", time.Second)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("health check: %v", err)
	}
	created, err := client.CreateResource(context.Background(), Resource{Name: "widget", Value: "desired"})
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if created.ID != "created-1" {
		t.Fatalf("created resource ID %q, want created-1", created.ID)
	}
	updated, err := client.UpdateResource(context.Background(), Resource{ID: created.ID, Name: created.Name, Value: "updated"})
	if err != nil {
		t.Fatalf("update resource: %v", err)
	}
	if updated.Value != "updated" {
		t.Fatalf("updated resource value %q, want updated", updated.Value)
	}
	if err := client.DeleteResource(context.Background(), "widget"); err != nil {
		t.Fatalf("delete resource: %v", err)
	}
}

func TestClientReportsExternalErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "upstream failed", http.StatusBadGateway)
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", time.Second)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if err := client.Health(context.Background()); err == nil || !strings.Contains(err.Error(), "502 Bad Gateway") || !strings.Contains(err.Error(), "upstream failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRejectsInvalidEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		token    string
	}{
		{name: "missing host", endpoint: "https://", token: "test-token"},
		{name: "unsupported scheme", endpoint: "ftp://example.test", token: "test-token"},
		{name: "missing token", endpoint: "https://example.test", token: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.endpoint, test.token, time.Second); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func writeTestJSON(writer http.ResponseWriter, status int, value Resource) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
