package mockexternalapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/winrarr/operator-foundry/internal/exampleclient"
)

func TestServerSupportsReferenceClientLifecycle(t *testing.T) {
	server := httptest.NewServer(New().Handler())
	defer server.Close()

	client, err := exampleclient.New(server.URL, "test-token", 0)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("health: %v", err)
	}
	created, err := client.CreateResource(context.Background(), exampleclient.Resource{Name: "widget", Value: "one"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("create returned an empty ID")
	}
	updated, err := client.UpdateResource(context.Background(), exampleclient.Resource{Name: "widget", Value: "two"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.ID != created.ID || updated.Value != "two" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	if err := client.DeleteResource(context.Background(), "widget"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := client.GetResource(context.Background(), "widget"); err != exampleclient.ErrNotFound {
		t.Fatalf("get after delete error = %v, want ErrNotFound", err)
	}
}

func TestServerAdminSurfaceSeedsAndInjectsFailures(t *testing.T) {
	server := httptest.NewServer(New().Handler())
	defer server.Close()
	httpClient := server.Client()

	seedBody := strings.NewReader(`{"id":"adopted-1","name":"ignored","value":"ready"}`)
	request, err := http.NewRequest(http.MethodPost, server.URL+"/admin/resources/adopted", seedBody)
	if err != nil {
		t.Fatalf("build seed request: %v", err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		t.Fatalf("seed resource: %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("seed status %s, want 201", response.Status)
	}
	_ = response.Body.Close()

	failureBody := strings.NewReader(`{"count":1,"status":503}`)
	request, err = http.NewRequest(http.MethodPost, server.URL+"/admin/fail-next", failureBody)
	if err != nil {
		t.Fatalf("build failure request: %v", err)
	}
	response, err = httpClient.Do(request)
	if err != nil {
		t.Fatalf("configure failure: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("failure configuration status %s, want 204", response.Status)
	}
	_ = response.Body.Close()

	request, err = http.NewRequest(http.MethodGet, server.URL+"/resources/adopted", nil)
	if err != nil {
		t.Fatalf("build external request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer test-token")
	response, err = httpClient.Do(request)
	if err != nil {
		t.Fatalf("external request: %v", err)
	}
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("injected failure status %s, want 503", response.Status)
	}
	_ = response.Body.Close()

	response, err = httpClient.Get(server.URL + "/admin/resources/adopted")
	if err != nil {
		t.Fatalf("get seeded resource: %v", err)
	}
	var resource exampleclient.Resource
	if err := json.NewDecoder(response.Body).Decode(&resource); err != nil {
		t.Fatalf("decode seeded resource: %v", err)
	}
	_ = response.Body.Close()
	if resource.ID != "adopted-1" || resource.Name != "adopted" {
		t.Fatalf("unexpected seeded resource: %#v", resource)
	}

	request, err = http.NewRequest(http.MethodPost, server.URL+"/admin/clear-failures", nil)
	if err != nil {
		t.Fatalf("build clear failures request: %v", err)
	}
	response, err = httpClient.Do(request)
	if err != nil {
		t.Fatalf("clear failures request: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("clear failures status %s, want 204", response.Status)
	}
	_ = response.Body.Close()
}

func TestParseStatsAndFormatStats(t *testing.T) {
	stats, err := ParseStats([]byte(`{"health":1,"get":2,"create":3,"update":4,"delete":5}`))
	if err != nil {
		t.Fatalf("parse stats: %v", err)
	}
	if got := FormatStats(stats); got != "health=1 get=2 create=3 update=4 delete=5" {
		t.Fatalf("formatted stats %q", got)
	}
	if _, err := ParseStats([]byte(`{"get":-1}`)); err == nil {
		t.Fatal("expected negative stats to fail")
	}
}
