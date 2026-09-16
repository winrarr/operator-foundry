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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// PatternResourceSpec demonstrates the common lifecycle fields of an external resource.
type PatternResourceSpec struct {
	// ConnectionRef identifies the connection used for external operations.
	ConnectionRef LocalObjectReference `json:"connectionRef"`
	// ExternalName is the external identity. When omitted, metadata.name is used.
	// +optional
	ExternalName string `json:"externalName,omitempty"`
	// Value is the desired external value.
	Value string `json:"value"`
	// CreationPolicy controls create versus adoption behavior.
	// Defaults to Create.
	// +optional
	CreationPolicy CreationPolicy `json:"creationPolicy,omitempty"`
	// DeletionPolicy controls whether deleting this object deletes its external resource.
	// Defaults to Orphan.
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
	// DriftDetectionInterval controls periodic external observations.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`
}

// PatternResourceStatus records the external identity and last observation.
type PatternResourceStatus struct {
	StatusBase `json:",inline"`
	// ID is the stable external identifier.
	// +optional
	ID string `json:"id,omitempty"`
	// ObservedValue is the last value read from the external system.
	// +optional
	ObservedValue string `json:"observedValue,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=pres
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="External ID",type=string,JSONPath=`.status.id`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type PatternResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PatternResourceSpec   `json:"spec,omitempty"`
	Status PatternResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PatternResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PatternResource `json:"items"`
}
