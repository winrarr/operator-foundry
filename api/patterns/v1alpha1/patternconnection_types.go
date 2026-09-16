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

// PatternConnectionSpec configures access to the example external API.
type PatternConnectionSpec struct {
	// Endpoint is the external API base URL.
	// +kubebuilder:validation:Pattern=`^https?://`
	Endpoint string `json:"endpoint"`
	// AuthSecretRef references a same-namespace Secret containing the bearer token.
	AuthSecretRef SecretKeyReference `json:"authSecretRef"`
	// RequestTimeout bounds each external API request.
	// +optional
	RequestTimeout *metav1.Duration `json:"requestTimeout,omitempty"`
}

// PatternConnectionStatus reports external reachability.
type PatternConnectionStatus struct {
	StatusBase `json:",inline"`
	// LastCheckedAt is the time of the most recent reachability check.
	// +optional
	LastCheckedAt *metav1.Time `json:"lastCheckedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=pconn
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type PatternConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PatternConnectionSpec   `json:"spec,omitempty"`
	Status PatternConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PatternConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PatternConnection `json:"items"`
}
