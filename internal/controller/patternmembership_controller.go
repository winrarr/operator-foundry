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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	patternsv1alpha1 "github.com/winrarr/operator-foundry/api/patterns/v1alpha1"
	"github.com/winrarr/operator-foundry/internal/exampleclient"
)

// PatternMembershipReconciler demonstrates ownership of one external
// relationship edge between two PatternResources.
type PatternMembershipReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternmemberships,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternmemberships/finalizers,verbs=update
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternmemberships/status,verbs=get;patch;update
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternresources,verbs=get;list;watch
// +kubebuilder:rbac:groups=patterns.operator-foundry.example,resources=patternconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *PatternMembershipReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var membership patternsv1alpha1.PatternMembership
	if err := r.Get(ctx, req.NamespacedName, &membership); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !membership.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, &membership)
	}
	if err := ensureFinalizer(ctx, r.Client, &membership); err != nil {
		return ctrl.Result{}, err
	}

	before := membership.DeepCopy()
	membership.Status.ObservedGeneration = membership.Generation
	connection, err := getConnection(ctx, r.Client, membership.Namespace, membership.Spec.ConnectionRef.Name)
	if err != nil {
		return r.recordMembershipDependency(ctx, &membership, before, err)
	}
	if !meta.IsStatusConditionTrue(connection.Status.Conditions, ConditionReady) {
		err := newDependencyError("PatternConnection %s/%s is not ready", connection.Namespace, connection.Name)
		return r.recordMembershipDependency(ctx, &membership, before, err)
	}

	parent, err := r.getResource(ctx, membership.Namespace, membership.Spec.ParentRef.Name)
	if err != nil {
		return r.recordMembershipDependency(ctx, &membership, before, err)
	}
	member, err := r.getResource(ctx, membership.Namespace, membership.Spec.MemberRef.Name)
	if err != nil {
		return r.recordMembershipDependency(ctx, &membership, before, err)
	}
	if parent.Spec.ConnectionRef.Name != connection.Name || member.Spec.ConnectionRef.Name != connection.Name {
		err := fmt.Errorf("parent and member PatternResources must use PatternConnection %s/%s", connection.Namespace, connection.Name)
		return r.recordMembershipStalled(ctx, &membership, before, "ConnectionMismatch", err)
	}
	if !meta.IsStatusConditionTrue(parent.Status.Conditions, ConditionReady) || parent.Status.ID == "" {
		err := newDependencyError("parent PatternResource %s/%s is not ready", parent.Namespace, parent.Name)
		return r.recordMembershipDependency(ctx, &membership, before, err)
	}
	if !meta.IsStatusConditionTrue(member.Status.Conditions, ConditionReady) || member.Status.ID == "" {
		err := newDependencyError("member PatternResource %s/%s is not ready", member.Namespace, member.Name)
		return r.recordMembershipDependency(ctx, &membership, before, err)
	}
	if membership.Status.ParentID != "" && membership.Status.ParentID != parent.Status.ID {
		err := fmt.Errorf("parent external identity changed from %q to %q; delete and recreate the PatternMembership", membership.Status.ParentID, parent.Status.ID)
		return r.recordMembershipStalled(ctx, &membership, before, "ExternalIdentityChanged", err)
	}
	if membership.Status.MemberID != "" && membership.Status.MemberID != member.Status.ID {
		err := fmt.Errorf("member external identity changed from %q to %q; delete and recreate the PatternMembership", membership.Status.MemberID, member.Status.ID)
		return r.recordMembershipStalled(ctx, &membership, before, "ExternalIdentityChanged", err)
	}

	apiClient, err := externalClientForConnection(ctx, r.Client, connection)
	if err != nil {
		if isDependencyError(err) {
			return r.recordMembershipDependency(ctx, &membership, before, err)
		}
		return r.recordMembershipStalled(ctx, &membership, before, "InvalidConfiguration", err)
	}
	if err := apiClient.EnsureMembership(ctx, parent.Status.ID, member.Status.ID); err != nil {
		return r.recordMembershipError(ctx, &membership, before, "ExternalRequestFailed", err)
	}

	membership.Status.ParentID = parent.Status.ID
	membership.Status.MemberID = member.Status.ID
	markReady(&membership.Status.Conditions, membership.Generation, "external relationship edge is reconciled")
	if err := persistStatus(ctx, r.Client, &membership, before); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: requeueFor(membership.Spec.DriftDetectionInterval, 0)}, nil
}

func (r *PatternMembershipReconciler) reconcileDeletion(ctx context.Context, membership *patternsv1alpha1.PatternMembership) (ctrl.Result, error) {
	if membership.Status.ParentID == "" || membership.Status.MemberID == "" {
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, membership)
	}
	connection, err := getConnection(ctx, r.Client, membership.Namespace, membership.Spec.ConnectionRef.Name)
	if err != nil {
		return removeFinalizerAfterDependencyLoss(ctx, r.Client, membership, "PatternConnection", err)
	}
	apiClient, err := externalClientForConnection(ctx, r.Client, connection)
	if err != nil {
		return removeFinalizerAfterDependencyLoss(ctx, r.Client, membership, "PatternConnection credentials", err)
	}
	if err := apiClient.DeleteMembership(ctx, membership.Status.ParentID, membership.Status.MemberID); err != nil && !errors.Is(err, exampleclient.ErrNotFound) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, removeFinalizer(ctx, r.Client, membership)
}

func (r *PatternMembershipReconciler) getResource(ctx context.Context, namespace, name string) (*patternsv1alpha1.PatternResource, error) {
	if name == "" {
		return nil, newDependencyError("PatternMembership resource reference is required")
	}
	var resource patternsv1alpha1.PatternResource
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &resource); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newDependencyError("PatternResource %s/%s was not found", namespace, name)
		}
		return nil, fmt.Errorf("get PatternResource %s/%s: %w", namespace, name, err)
	}
	return &resource, nil
}

func (r *PatternMembershipReconciler) recordMembershipDependency(ctx context.Context, membership *patternsv1alpha1.PatternMembership, before client.Object, err error) (ctrl.Result, error) {
	markReconciling(&membership.Status.Conditions, membership.Generation, "DependencyNotReady", err.Error())
	if statusErr := persistStatus(ctx, r.Client, membership, before); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: DependencyRetry}, nil
}

func (r *PatternMembershipReconciler) recordMembershipStalled(ctx context.Context, membership *patternsv1alpha1.PatternMembership, before client.Object, reason string, err error) (ctrl.Result, error) {
	markStalled(&membership.Status.Conditions, membership.Generation, reason, err)
	return ctrl.Result{}, persistStatus(ctx, r.Client, membership, before)
}

func (r *PatternMembershipReconciler) recordMembershipError(ctx context.Context, membership *patternsv1alpha1.PatternMembership, before client.Object, reason string, err error) (ctrl.Result, error) {
	markError(&membership.Status.Conditions, membership.Generation, reason, err)
	if statusErr := persistStatus(ctx, r.Client, membership, before); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: ExternalRetry}, nil
}

func (r *PatternMembershipReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&patternsv1alpha1.PatternMembership{}).
		Watches(&patternsv1alpha1.PatternResource{}, handler.EnqueueRequestsFromMapFunc(r.mapResourceToMemberships)).
		Watches(&patternsv1alpha1.PatternConnection{}, handler.EnqueueRequestsFromMapFunc(r.mapConnectionToMemberships)).
		Complete(r)
}

func (r *PatternMembershipReconciler) mapResourceToMemberships(ctx context.Context, object client.Object) []reconcile.Request {
	var memberships patternsv1alpha1.PatternMembershipList
	if err := r.List(ctx, &memberships, client.InNamespace(object.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range memberships.Items {
		membership := &memberships.Items[i]
		if membership.Spec.ParentRef.Name == object.GetName() || membership.Spec.MemberRef.Name == object.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(membership)})
		}
	}
	return requests
}

func (r *PatternMembershipReconciler) mapConnectionToMemberships(ctx context.Context, object client.Object) []reconcile.Request {
	var memberships patternsv1alpha1.PatternMembershipList
	if err := r.List(ctx, &memberships, client.InNamespace(object.GetNamespace())); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	for i := range memberships.Items {
		membership := &memberships.Items[i]
		if membership.Spec.ConnectionRef.Name == object.GetName() {
			requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(membership)})
		}
	}
	return requests
}
