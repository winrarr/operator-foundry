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

package controller

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	patternsv1alpha1 "github.com/winrarr/operator-foundry/api/patterns/v1alpha1"
	"github.com/winrarr/operator-foundry/internal/exampleclient"
)

// PatternResourceReconciler demonstrates the common external-resource
// lifecycle. It is intentionally small enough to adapt rather than a framework.
type PatternResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternresources,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternresources/finalizers,verbs=update
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternresources/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *PatternResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var resource patternsv1alpha1.PatternResource
	if err := r.Get(ctx, req.NamespacedName, &resource); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !resource.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &resource)
	}
	if resource.Spec.DeletionPolicy.Effective() == patternsv1alpha1.DeletionPolicyDelete {
		if err := ensureFinalizer(ctx, r.Client, &resource); err != nil {
			return ctrl.Result{}, err
		}
	}

	before := resource.DeepCopy()
	resource.Status.ObservedGeneration = resource.Generation
	connection, err := getConnection(ctx, r.Client, resource.Namespace, resource.Spec.ConnectionRef.Name)
	if err != nil {
		return r.recordDependency(ctx, &resource, before, err)
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, ConditionReady) {
		err := newDependencyError("PatternConnection %s/%s is not ready", connection.Namespace, connection.Name)
		return r.recordDependency(ctx, &resource, before, err)
	}

	apiClient, err := externalClientForConnection(ctx, r.Client, connection)
	if err != nil {
		if isDependencyError(err) {
			return r.recordDependency(ctx, &resource, before, err)
		}
		return r.recordStalled(ctx, &resource, before, "InvalidConfiguration", err)
	}

	name := externalName(resource.Name, resource.Spec.ExternalName)
	existing, err := apiClient.GetResource(ctx, name)
	if errors.Is(err, exampleclient.ErrNotFound) {
		if !resource.Spec.CreationPolicy.Effective().AllowsCreation() {
			return r.recordStalled(ctx, &resource, before, "CreationNotAllowed", fmt.Errorf("external resource %q does not exist and creationPolicy=%s does not allow creation", name, resource.Spec.CreationPolicy.Effective()))
		}
		existing, err = apiClient.CreateResource(ctx, exampleclient.Resource{Name: name, Value: resource.Spec.Value})
	} else if err == nil {
		if resource.Status.ID != "" && existing.ID != resource.Status.ID {
			return r.recordStalled(ctx, &resource, before, "ExternalIdentityChanged", fmt.Errorf("external resource identity changed from %q to %q", resource.Status.ID, existing.ID))
		}
		if resource.Status.ID == "" && !resource.Spec.CreationPolicy.Effective().AllowsAdoption() {
			return r.recordStalled(ctx, &resource, before, "ExternalResourceExists", fmt.Errorf("external resource %q already exists; set creationPolicy to Adopt or CreateOrAdopt to manage it", name))
		}
		if existing.Value != resource.Spec.Value {
			existing, err = apiClient.UpdateResource(ctx, exampleclient.Resource{ID: existing.ID, Name: name, Value: resource.Spec.Value})
		}
	}
	if err != nil {
		if errors.Is(err, exampleclient.ErrNotFound) {
			return r.recordStalled(ctx, &resource, before, "ExternalResourceMissing", err)
		}
		return r.recordError(ctx, &resource, before, "ExternalRequestFailed", err)
	}

	resource.Status.ID = existing.ID
	resource.Status.ObservedValue = existing.Value
	markReady(&resource.Status.Conditions, resource.Generation, "external resource is reconciled")
	if err := persistStatus(ctx, r.Client, &resource, before); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: requeueFor(resource.Spec.DriftDetectionInterval, 0)}, nil
}

func (r *PatternResourceReconciler) reconcileDeletion(ctx context.Context, resource *patternsv1alpha1.PatternResource) (ctrl.Result, error) {
	if resource.Spec.DeletionPolicy.Effective() != patternsv1alpha1.DeletionPolicyDelete || resource.Status.ID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, resource)
	}
	connection, err := getConnection(ctx, r.Client, resource.Namespace, resource.Spec.ConnectionRef.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	apiClient, err := externalClientForConnection(ctx, r.Client, connection)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := apiClient.DeleteResource(ctx, externalName(resource.Name, resource.Spec.ExternalName)); err != nil && !errors.Is(err, exampleclient.ErrNotFound) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, resource)
}

func (r *PatternResourceReconciler) recordDependency(ctx context.Context, resource *patternsv1alpha1.PatternResource, before client.Object, err error) (ctrl.Result, error) {
	markReconciling(&resource.Status.Conditions, resource.Generation, "DependencyNotReady", err.Error())
	if statusErr := persistStatus(ctx, r.Client, resource, before); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: DependencyRetry}, nil
}

func (r *PatternResourceReconciler) recordStalled(ctx context.Context, resource *patternsv1alpha1.PatternResource, before client.Object, reason string, err error) (ctrl.Result, error) {
	markStalled(&resource.Status.Conditions, resource.Generation, reason, err)
	return ctrl.Result{}, persistStatus(ctx, r.Client, resource, before)
}

func (r *PatternResourceReconciler) recordError(ctx context.Context, resource *patternsv1alpha1.PatternResource, before client.Object, reason string, err error) (ctrl.Result, error) {
	markError(&resource.Status.Conditions, resource.Generation, reason, err)
	if statusErr := persistStatus(ctx, r.Client, resource, before); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: ExternalRetry}, nil
}

func (r *PatternResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&patternsv1alpha1.PatternResource{}).
		Watches(&patternsv1alpha1.PatternConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToResources)).
		Complete(r)
}

func (r *PatternResourceReconciler) mapConnectionToResources(ctx context.Context, object client.Object) []reconcile.Request {
	var resources patternsv1alpha1.PatternResourceList
	if err := r.List(ctx, &resources, client.InNamespace(object.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range resources.Items {
		resource := &resources.Items[i]
		if resource.Spec.ConnectionRef.Name == object.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(resource)})
		}
	}
	return requests
}
