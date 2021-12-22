/*
Copyright 2021 NDD.

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
	"reflect"

	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// NiPoolFinalizer is the name of the finalizer added to
	// NiPool to block delete operations until the physical node can be
	// deprovisioned.
	NiPoolFinalizer string = "niPool.nipool.nddr.yndd.io"
)

// NiPool struct
type NipoolNiPool struct {
	// +kubebuilder:validation:Enum=`disable`;`enable`
	// +kubebuilder:default:="enable"
	AdminState *string `json:"admin-state,omitempty"`
	// +kubebuilder:validation:Enum=`first-available`
	// +kubebuilder:default:="first-available"
	AllocationStrategy *string `json:"allocation-strategy,omitempty"`
	// kubebuilder:validation:Minimum=1
	// kubebuilder:validation:Maximum=8192
	Size *uint32 `json:"size"`
	// kubebuilder:validation:MinLength=1
	// kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="[A-Za-z0-9 !@#$^&()|+=`~.,'/_:;?-]*"
	Description *string `json:"description,omitempty"`
}

// A NiPoolSpec defines the desired state of a NiPool.
type NiPoolSpec struct {
	//nddv1.ResourceSpec `json:",inline"`
	NipoolNiPool *NipoolNiPool `json:"ni-pool,omitempty"`
}

// A NiPoolStatus represents the observed state of a NiPool.
type NiPoolStatus struct {
	nddv1.ConditionedStatus `json:",inline"`
	NipoolNiPool            *NddrNiPoolNiPool `json:"ni-pool,omitempty"`
}

// +kubebuilder:object:root=true

// NiPool is the Schema for the NiPool API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SYNC",type="string",JSONPath=".status.conditions[?(@.kind=='Synced')].status"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.conditions[?(@.kind=='Ready')].status"
// +kubebuilder:printcolumn:name="ALLOCATED",type="string",JSONPath=".status.ni-pool.state.allocated",description="allocated network-instance"
// +kubebuilder:printcolumn:name="AVAILABLE",type="string",JSONPath=".status.ni-pool.state.available",description="available network-instance"
// +kubebuilder:printcolumn:name="TOTAL",type="string",JSONPath=".status.ni-pool.state.total",description="total network-instances"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type NiPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NiPoolSpec   `json:"spec,omitempty"`
	Status NiPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NiPoolList contains a list of NiPools
type NiPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NiPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NiPool{}, &NiPoolList{})
}

// NiPool type metadata.
var (
	NiPoolKindKind         = reflect.TypeOf(NiPool{}).Name()
	NiPoolGroupKind        = schema.GroupKind{Group: Group, Kind: NiPoolKindKind}.String()
	NiPoolKindAPIVersion   = NiPoolKindKind + "." + GroupVersion.String()
	NiPoolGroupVersionKind = GroupVersion.WithKind(NiPoolKindKind)
)
