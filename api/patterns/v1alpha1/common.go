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

// CreationPolicy controls how an external resource is acquired.
// +kubebuilder:validation:Enum=Create;Adopt;CreateOrAdopt
type CreationPolicy string

const (
	CreationPolicyCreate        CreationPolicy = "Create"
	CreationPolicyAdopt         CreationPolicy = "Adopt"
	CreationPolicyCreateOrAdopt CreationPolicy = "CreateOrAdopt"
)

func (policy CreationPolicy) AllowsCreation() bool {
	return policy == "" || policy == CreationPolicyCreate || policy == CreationPolicyCreateOrAdopt
}

func (policy CreationPolicy) AllowsAdoption() bool {
	return policy == CreationPolicyAdopt || policy == CreationPolicyCreateOrAdopt
}

// DeletionPolicy controls whether the external resource is deleted with the Kubernetes resource.
// +kubebuilder:validation:Enum=Delete;Orphan
type DeletionPolicy string

const (
	DeletionPolicyDelete DeletionPolicy = "Delete"
	DeletionPolicyOrphan DeletionPolicy = "Orphan"
)

// SecretKeyReference identifies a key in a Secret in the same namespace.
type SecretKeyReference struct {
	// Name is the Secret name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Key is the data key. It defaults to token when omitted.
	// +optional
	Key string `json:"key,omitempty"`
}

// LocalObjectReference identifies a resource in the same namespace.
type LocalObjectReference struct {
	// Name is the referenced resource name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// StatusBase is the common status shape demonstrated by the reference resources.
type StatusBase struct {
	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (policy CreationPolicy) Effective() CreationPolicy {
	if policy == "" {
		return CreationPolicyCreate
	}
	return policy
}

func (policy DeletionPolicy) Effective() DeletionPolicy {
	if policy == "" {
		return DeletionPolicyOrphan
	}
	return policy
}
