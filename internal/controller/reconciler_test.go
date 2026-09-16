package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	patternsv1alpha1 "github.com/winrarr/operator-foundry/api/patterns/v1alpha1"
	"github.com/winrarr/operator-foundry/internal/exampleclient"
)

type externalTestServer struct {
	mu        sync.Mutex
	resources map[string]exampleclient.Resource
	requests  []string
	server    *httptest.Server
}

func newExternalTestServer(t *testing.T, resources ...exampleclient.Resource) *externalTestServer {
	t.Helper()
	external := &externalTestServer{resources: make(map[string]exampleclient.Resource)}
	for _, resource := range resources {
		external.resources[resource.Name] = resource
	}
	external.server = httptest.NewServer(http.HandlerFunc(external.serveHTTP))
	t.Cleanup(external.server.Close)
	return external
}

func (e *externalTestServer) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Authorization") != "Bearer test-token" {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}
	e.mu.Lock()
	e.requests = append(e.requests, request.Method+" "+request.URL.Path)
	e.mu.Unlock()

	if request.URL.Path == "/health" && request.Method == http.MethodGet {
		writer.WriteHeader(http.StatusOK)
		return
	}
	if !strings.HasPrefix(request.URL.Path, "/resources/") && request.URL.Path != "/resources" {
		http.NotFound(writer, request)
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	name := strings.TrimPrefix(request.URL.Path, "/resources/")
	if request.Method == http.MethodPost {
		var resource exampleclient.Resource
		if err := json.NewDecoder(request.Body).Decode(&resource); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		resource.ID = "created-" + resource.Name
		e.resources[resource.Name] = resource
		writeJSON(writer, http.StatusCreated, resource)
		return
	}

	resource, exists := e.resources[name]
	if !exists {
		http.NotFound(writer, request)
		return
	}
	switch request.Method {
	case http.MethodGet:
		writeJSON(writer, http.StatusOK, resource)
	case http.MethodPut:
		var desired exampleclient.Resource
		if err := json.NewDecoder(request.Body).Decode(&desired); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		desired.ID = resource.ID
		e.resources[name] = desired
		writeJSON(writer, http.StatusOK, desired)
	case http.MethodDelete:
		delete(e.resources, name)
		writer.WriteHeader(http.StatusNoContent)
	default:
		writer.Header().Set("Allow", "GET, PUT, DELETE")
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func newControllerTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := patternsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add pattern scheme: %v", err)
	}
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&patternsv1alpha1.PatternConnection{}, &patternsv1alpha1.PatternResource{}).
		WithObjects(objects...).
		Build()
}

func testConnection(endpoint string) *patternsv1alpha1.PatternConnection {
	return &patternsv1alpha1.PatternConnection{
		ObjectMeta: metav1.ObjectMeta{Name: "connection", Namespace: "demo", Generation: 1},
		Spec: patternsv1alpha1.PatternConnectionSpec{
			Endpoint:      endpoint,
			AuthSecretRef: patternsv1alpha1.SecretKeyReference{Name: "credentials"},
		},
		Status: patternsv1alpha1.PatternConnectionStatus{
			StatusBase: patternsv1alpha1.StatusBase{
				Conditions: []metav1.Condition{{
					Type:               ConditionReady,
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 1,
				}},
			},
		},
	}
}

func testCredentials() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "demo"},
		Data:       map[string][]byte{"token": []byte("test-token")},
	}
}

func reconcileRequest(name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: "demo"}}
}

func TestPatternConnectionReconcileRecordsMissingSecret(t *testing.T) {
	kubeClient := newControllerTestClient(t, testConnection("https://api.example.test"))
	reconciler := &PatternConnectionReconciler{Client: kubeClient}

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("connection"))
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.RequeueAfter != DependencyRetry {
		t.Fatalf("requeue after %s, want %s", result.RequeueAfter, DependencyRetry)
	}

	var connection patternsv1alpha1.PatternConnection
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "connection", Namespace: "demo"}, &connection); err != nil {
		t.Fatalf("get connection: %v", err)
	}
	assertConditionReason(t, connection.Status.Conditions, ConditionReady, "DependencyNotReady")
	if connection.Status.ObservedGeneration != 1 {
		t.Fatalf("observed generation %d, want 1", connection.Status.ObservedGeneration)
	}
}

func TestPatternConnectionReconcileRecordsReachability(t *testing.T) {
	external := newExternalTestServer(t)
	kubeClient := newControllerTestClient(t, testConnection(external.server.URL), testCredentials())
	reconciler := &PatternConnectionReconciler{Client: kubeClient}

	result, err := reconciler.Reconcile(context.Background(), reconcileRequest("connection"))
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result != (reconcile.Result{}) {
		t.Fatalf("result %#v, want empty result", result)
	}

	var connection patternsv1alpha1.PatternConnection
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "connection", Namespace: "demo"}, &connection); err != nil {
		t.Fatalf("get connection: %v", err)
	}
	assertConditionStatus(t, connection.Status.Conditions, ConditionReady, metav1.ConditionTrue)
	if connection.Status.LastCheckedAt == nil {
		t.Fatal("LastCheckedAt was not recorded")
	}
	if connection.Status.ObservedGeneration != 1 {
		t.Fatalf("observed generation %d, want 1", connection.Status.ObservedGeneration)
	}
}

func TestPatternResourceReconcileCreatesAndRecordsExternalResource(t *testing.T) {
	external := newExternalTestServer(t)
	resource := &patternsv1alpha1.PatternResource{
		ObjectMeta: metav1.ObjectMeta{Name: "widget", Namespace: "demo", Generation: 1},
		Spec: patternsv1alpha1.PatternResourceSpec{
			ConnectionRef: patternsv1alpha1.LocalObjectReference{Name: "connection"},
			Value:         "desired",
		},
	}
	kubeClient := newControllerTestClient(t, testConnection(external.server.URL), testCredentials(), resource)
	reconciler := &PatternResourceReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), reconcileRequest("widget")); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var observed patternsv1alpha1.PatternResource
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "widget", Namespace: "demo"}, &observed); err != nil {
		t.Fatalf("get resource: %v", err)
	}
	assertConditionStatus(t, observed.Status.Conditions, ConditionReady, metav1.ConditionTrue)
	if observed.Status.ID != "created-widget" || observed.Status.ObservedValue != "desired" {
		t.Fatalf("unexpected status: %#v", observed.Status)
	}
	if observed.Status.ObservedGeneration != 1 {
		t.Fatalf("observed generation %d, want 1", observed.Status.ObservedGeneration)
	}
}

func TestPatternResourceReconcileRequiresExplicitAdoption(t *testing.T) {
	external := newExternalTestServer(t, exampleclient.Resource{ID: "existing-1", Name: "widget", Value: "external"})
	resource := &patternsv1alpha1.PatternResource{
		ObjectMeta: metav1.ObjectMeta{Name: "widget", Namespace: "demo", Generation: 1},
		Spec: patternsv1alpha1.PatternResourceSpec{
			ConnectionRef: patternsv1alpha1.LocalObjectReference{Name: "connection"},
			Value:         "external",
		},
	}
	kubeClient := newControllerTestClient(t, testConnection(external.server.URL), testCredentials(), resource)
	reconciler := &PatternResourceReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), reconcileRequest("widget")); err != nil {
		t.Fatalf("reconcile without adoption: %v", err)
	}
	var observed patternsv1alpha1.PatternResource
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "widget", Namespace: "demo"}, &observed); err != nil {
		t.Fatalf("get resource: %v", err)
	}
	assertConditionReason(t, observed.Status.Conditions, ConditionReady, "ExternalResourceExists")

	observed.Spec.CreationPolicy = patternsv1alpha1.CreationPolicyAdopt
	if err := kubeClient.Update(context.Background(), &observed); err != nil {
		t.Fatalf("enable adoption: %v", err)
	}
	if _, err := reconciler.Reconcile(context.Background(), reconcileRequest("widget")); err != nil {
		t.Fatalf("reconcile with adoption: %v", err)
	}
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "widget", Namespace: "demo"}, &observed); err != nil {
		t.Fatalf("get adopted resource: %v", err)
	}
	assertConditionStatus(t, observed.Status.Conditions, ConditionReady, metav1.ConditionTrue)
	if observed.Status.ID != "existing-1" {
		t.Fatalf("adopted external ID %q, want existing-1", observed.Status.ID)
	}
}

func TestPatternResourceReconcileUpdatesKnownExternalResource(t *testing.T) {
	external := newExternalTestServer(t, exampleclient.Resource{ID: "existing-1", Name: "widget", Value: "old"})
	resource := &patternsv1alpha1.PatternResource{
		ObjectMeta: metav1.ObjectMeta{Name: "widget", Namespace: "demo", Generation: 1},
		Spec: patternsv1alpha1.PatternResourceSpec{
			ConnectionRef: patternsv1alpha1.LocalObjectReference{Name: "connection"},
			Value:         "new",
		},
		Status: patternsv1alpha1.PatternResourceStatus{
			StatusBase: patternsv1alpha1.StatusBase{Conditions: []metav1.Condition{{Type: ConditionReady, Status: metav1.ConditionTrue}}},
			ID:         "existing-1",
		},
	}
	kubeClient := newControllerTestClient(t, testConnection(external.server.URL), testCredentials(), resource)
	reconciler := &PatternResourceReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), reconcileRequest("widget")); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var observed patternsv1alpha1.PatternResource
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "widget", Namespace: "demo"}, &observed); err != nil {
		t.Fatalf("get resource: %v", err)
	}
	if observed.Status.ObservedValue != "new" {
		t.Fatalf("observed value %q, want new", observed.Status.ObservedValue)
	}
	if !containsRequest(external, http.MethodPut+" /resources/widget") {
		t.Fatalf("expected an update request, got %v", external.requests)
	}
}

func TestPatternResourceReconcileDeletesManagedExternalResource(t *testing.T) {
	external := newExternalTestServer(t, exampleclient.Resource{ID: "existing-1", Name: "widget", Value: "old"})
	deletionTime := metav1.NewTime(time.Now())
	resource := &patternsv1alpha1.PatternResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "widget",
			Namespace:         "demo",
			Generation:        1,
			DeletionTimestamp: &deletionTime,
			Finalizers:        []string{FinalizerName},
		},
		Spec: patternsv1alpha1.PatternResourceSpec{
			ConnectionRef:  patternsv1alpha1.LocalObjectReference{Name: "connection"},
			DeletionPolicy: patternsv1alpha1.DeletionPolicyDelete,
		},
		Status: patternsv1alpha1.PatternResourceStatus{ID: "existing-1"},
	}
	kubeClient := newControllerTestClient(t, testConnection(external.server.URL), testCredentials(), resource)
	reconciler := &PatternResourceReconciler{Client: kubeClient}

	if _, err := reconciler.Reconcile(context.Background(), reconcileRequest("widget")); err != nil {
		t.Fatalf("reconcile deletion: %v", err)
	}
	if !containsRequest(external, http.MethodDelete+" /resources/widget") {
		t.Fatalf("expected a delete request, got %v", external.requests)
	}
	var observed patternsv1alpha1.PatternResource
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Name: "widget", Namespace: "demo"}, &observed); !apierrors.IsNotFound(err) {
		t.Fatalf("resource should be removed after the finalizer is cleared, got %v", err)
	}
}

func containsRequest(external *externalTestServer, expected string) bool {
	external.mu.Lock()
	defer external.mu.Unlock()
	for _, request := range external.requests {
		if request == expected {
			return true
		}
	}
	return false
}

func assertConditionReason(t *testing.T, conditions []metav1.Condition, conditionType, expected string) {
	t.Helper()
	for _, condition := range conditions {
		if condition.Type == conditionType {
			if condition.Reason != expected {
				t.Fatalf("condition %s reason %s, want %s", conditionType, condition.Reason, expected)
			}
			return
		}
	}
	t.Fatalf("condition %s not found in %#v", conditionType, conditions)
}
