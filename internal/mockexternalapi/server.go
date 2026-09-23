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

// Package mockexternalapi provides the disposable external system used by the
// live reference-operator tests. It is deliberately not a production server.
package mockexternalapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/winrarr/operator-foundry/internal/exampleclient"
)

const testToken = "test-token"

// Server is an in-memory external API with a small administrative surface for
// deterministic E2E setup and fault injection.
type Server struct {
	mu          sync.Mutex
	resources   map[string]exampleclient.Resource
	memberships map[string]exampleclient.Membership
	requests    RequestStats
	nextID      int
	failure     failurePlan
}

type failurePlan struct {
	count  int
	status int
}

// RequestStats records external API calls. Administrative requests are not
// included, so tests can distinguish controller creates from test setup.
type RequestStats struct {
	Health int `json:"health"`
	Get    int `json:"get"`
	Create int `json:"create"`
	Update int `json:"update"`
	Delete int `json:"delete"`
}

// New creates a mock API seeded with the supplied resources.
func New(resources ...exampleclient.Resource) *Server {
	server := &Server{resources: make(map[string]exampleclient.Resource), memberships: make(map[string]exampleclient.Membership)}
	for _, resource := range resources {
		server.resources[resource.Name] = resource
	}
	return server
}

// Handler returns the HTTP handler for the mock API.
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/admin/") {
		s.serveAdmin(writer, request)
		return
	}
	if request.URL.Path == "/health" && request.Method == http.MethodGet {
		s.mu.Lock()
		s.requests.Health++
		s.mu.Unlock()
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if request.Header.Get("Authorization") != "Bearer "+testToken {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}
	if status, failed := s.consumeFailure(); failed {
		http.Error(writer, "injected upstream failure", status)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/memberships/") {
		s.serveMembership(writer, request, strings.TrimPrefix(request.URL.Path, "/memberships/"))
		return
	}

	if request.URL.Path == "/resources" && request.Method == http.MethodPost {
		s.createResource(writer, request)
		return
	}
	if !strings.HasPrefix(request.URL.Path, "/resources/") {
		http.NotFound(writer, request)
		return
	}

	name, err := url.PathUnescape(strings.TrimPrefix(request.URL.Path, "/resources/"))
	if err != nil || strings.TrimSpace(name) == "" {
		http.Error(writer, "invalid resource name", http.StatusBadRequest)
		return
	}
	switch request.Method {
	case http.MethodGet:
		s.getResource(writer, name)
	case http.MethodPut:
		s.updateResource(writer, request, name)
	case http.MethodDelete:
		s.deleteResource(writer, name)
	default:
		writer.Header().Set("Allow", "GET, PUT, DELETE")
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) serveAdmin(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/admin/memberships" && request.Method == http.MethodGet {
		parentID := request.URL.Query().Get("parentID")
		memberID := request.URL.Query().Get("memberID")
		if parentID == "" || memberID == "" {
			http.Error(writer, "parentID and memberID are required", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		membership, exists := s.memberships[exampleclient.MembershipKey(parentID, memberID)]
		s.mu.Unlock()
		if !exists {
			http.NotFound(writer, request)
			return
		}
		writeJSON(writer, http.StatusOK, membership)
		return
	}
	if request.URL.Path == "/admin/stats" && request.Method == http.MethodGet {
		s.mu.Lock()
		stats := s.requests
		s.mu.Unlock()
		writeJSON(writer, http.StatusOK, stats)
		return
	}
	if request.URL.Path == "/admin/fail-next" && request.Method == http.MethodPost {
		var input struct {
			Count  int `json:"count"`
			Status int `json:"status"`
		}
		if err := decodeJSON(request, &input); err != nil || input.Count < 1 || input.Status < 400 || input.Status > 599 {
			http.Error(writer, "count must be positive and status must be 400-599", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.failure = failurePlan{count: input.Count, status: input.Status}
		s.mu.Unlock()
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if request.URL.Path == "/admin/clear-failures" && request.Method == http.MethodPost {
		s.mu.Lock()
		s.failure = failurePlan{}
		s.mu.Unlock()
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	if !strings.HasPrefix(request.URL.Path, "/admin/resources/") {
		http.NotFound(writer, request)
		return
	}
	name, err := url.PathUnescape(strings.TrimPrefix(request.URL.Path, "/admin/resources/"))
	if err != nil || strings.TrimSpace(name) == "" {
		http.Error(writer, "invalid resource name", http.StatusBadRequest)
		return
	}
	switch request.Method {
	case http.MethodGet:
		s.getAdminResource(writer, name)
	case http.MethodPost:
		s.seedResource(writer, request, name)
	case http.MethodDelete:
		s.removeAdminResource(writer, name)
	default:
		writer.Header().Set("Allow", "GET, POST, DELETE")
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) serveMembership(writer http.ResponseWriter, request *http.Request, key string) {
	if key == "" {
		http.Error(writer, "membership key is required", http.StatusBadRequest)
		return
	}
	switch request.Method {
	case http.MethodPut:
		var membership exampleclient.Membership
		if err := decodeJSON(request, &membership); err != nil || membership.ParentID == "" || membership.MemberID == "" {
			http.Error(writer, "parentID and memberID are required", http.StatusBadRequest)
			return
		}
		if key != exampleclient.MembershipKey(membership.ParentID, membership.MemberID) {
			http.Error(writer, "membership key does not match the edge", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.memberships[key] = membership
		s.mu.Unlock()
		writer.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		s.mu.Lock()
		_, exists := s.memberships[key]
		delete(s.memberships, key)
		s.mu.Unlock()
		if !exists {
			http.NotFound(writer, request)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	default:
		writer.Header().Set("Allow", "PUT, DELETE")
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) createResource(writer http.ResponseWriter, request *http.Request) {
	var resource exampleclient.Resource
	if err := decodeJSON(request, &resource); err != nil || strings.TrimSpace(resource.Name) == "" {
		http.Error(writer, "name is required", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.resources[resource.Name]; exists {
		writeError(writer, http.StatusConflict, "resource already exists")
		return
	}
	s.nextID++
	resource.ID = fmt.Sprintf("created-%s-%d", resource.Name, s.nextID)
	s.resources[resource.Name] = resource
	s.requests.Create++
	writeJSON(writer, http.StatusCreated, resource)
}

func (s *Server) getResource(writer http.ResponseWriter, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests.Get++
	resource, exists := s.resources[name]
	if !exists {
		http.NotFound(writer, nil)
		return
	}
	writeJSON(writer, http.StatusOK, resource)
}

func (s *Server) updateResource(writer http.ResponseWriter, request *http.Request, name string) {
	var desired exampleclient.Resource
	if err := decodeJSON(request, &desired); err != nil {
		http.Error(writer, "invalid resource body", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	resource, exists := s.resources[name]
	if !exists {
		http.NotFound(writer, nil)
		return
	}
	desired.ID = resource.ID
	desired.Name = name
	s.resources[name] = desired
	s.requests.Update++
	writeJSON(writer, http.StatusOK, desired)
}

func (s *Server) deleteResource(writer http.ResponseWriter, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.resources[name]; !exists {
		http.NotFound(writer, nil)
		return
	}
	delete(s.resources, name)
	s.requests.Delete++
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) getAdminResource(writer http.ResponseWriter, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resource, exists := s.resources[name]
	if !exists {
		http.NotFound(writer, nil)
		return
	}
	writeJSON(writer, http.StatusOK, resource)
}

func (s *Server) seedResource(writer http.ResponseWriter, request *http.Request, name string) {
	var resource exampleclient.Resource
	if err := decodeJSON(request, &resource); err != nil {
		http.Error(writer, "invalid resource body", http.StatusBadRequest)
		return
	}
	resource.Name = name
	if resource.ID == "" {
		resource.ID = "seeded-" + name
	}
	s.mu.Lock()
	s.resources[name] = resource
	s.mu.Unlock()
	writeJSON(writer, http.StatusCreated, resource)
}

func (s *Server) removeAdminResource(writer http.ResponseWriter, name string) {
	s.mu.Lock()
	delete(s.resources, name)
	s.mu.Unlock()
	writer.WriteHeader(http.StatusNoContent)
}

func (s *Server) consumeFailure() (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure.count == 0 {
		return 0, false
	}
	status := s.failure.status
	s.failure.count--
	if s.failure.count == 0 {
		s.failure = failurePlan{}
	}
	return status, true
}

func decodeJSON(request *http.Request, target any) error {
	defer func() { _ = request.Body.Close() }()
	return json.NewDecoder(request.Body).Decode(target)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	http.Error(writer, message, status)
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

// ParseStats is a small helper for shell tests that want to fail clearly on a
// malformed administrative response.
func ParseStats(data []byte) (RequestStats, error) {
	var stats RequestStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return RequestStats{}, err
	}
	if stats.Health < 0 || stats.Get < 0 || stats.Create < 0 || stats.Update < 0 || stats.Delete < 0 {
		return RequestStats{}, errors.New("request counts cannot be negative")
	}
	return stats, nil
}

// FormatStats is useful when embedding diagnostics in E2E failure output.
func FormatStats(stats RequestStats) string {
	return "health=" + strconv.Itoa(stats.Health) +
		" get=" + strconv.Itoa(stats.Get) +
		" create=" + strconv.Itoa(stats.Create) +
		" update=" + strconv.Itoa(stats.Update) +
		" delete=" + strconv.Itoa(stats.Delete)
}
