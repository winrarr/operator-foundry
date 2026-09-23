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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PatternMembershipSpec claims one relationship edge between two managed
// PatternResources. Delete the PatternMembership to delete only this edge.
// +kubebuilder:validation:XValidation:rule="self.connectionRef == oldSelf.connectionRef && self.parentRef == oldSelf.parentRef && self.memberRef == oldSelf.memberRef",message="membership references are immutable; delete and recreate the PatternMembership"
// +kubebuilder:validation:XValidation:rule="self.parentRef.name != self.memberRef.name",message="parentRef and memberRef must identify different PatternResources"
type PatternMembershipSpec struct {
	// ConnectionRef selects the same-namespace PatternConnection used for the edge.
	ConnectionRef LocalObjectReference `json:"connectionRef"`

	// ParentRef identifies the resource that owns the relationship.
	ParentRef LocalObjectReference `json:"parentRef"`

	// MemberRef identifies the resource connected to the parent.
	MemberRef LocalObjectReference `json:"memberRef"`

	// DriftDetectionInterval controls periodic re-observation of the external edge.
	// When omitted, the controller reconciles on Kubernetes events only.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`
}

// PatternMembershipStatus reports the stable external identities for this edge.
type PatternMembershipStatus struct {
	StatusBase `json:",inline"`

	// ParentID is the external identity of the parent resource.
	// +optional
	ParentID string `json:"parentID,omitempty"`

	// MemberID is the external identity of the member resource.
	// +optional
	MemberID string `json:"memberID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=pmem
// +kubebuilder:printcolumn:name="Parent ID",type=string,JSONPath=`.status.parentID`
// +kubebuilder:printcolumn:name="Member ID",type=string,JSONPath=`.status.memberID`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// PatternMembership claims one relationship edge between two PatternResources.
type PatternMembership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PatternMembershipSpec   `json:"spec"`
	Status PatternMembershipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// PatternMembershipList contains PatternMembership objects.
type PatternMembershipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PatternMembership `json:"items"`
}
