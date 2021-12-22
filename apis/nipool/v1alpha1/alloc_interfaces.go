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
	"fmt"
	"reflect"

	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	"github.com/yndd/ndd-runtime/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ AaList = &AllocList{}

// +k8s:deepcopy-gen=false
type AaList interface {
	client.ObjectList

	GetAllocs() []Aa
}

func (x *AllocList) GetAllocs() []Aa {
	allocs := make([]Aa, len(x.Items))
	for i, r := range x.Items {
		r := r // Pin range variable so we can take its address.
		allocs[i] = &r
	}
	return allocs
}

var _ Aa = &Alloc{}

// +k8s:deepcopy-gen=false
type Aa interface {
	resource.Object
	resource.Conditioned

	GetCondition(ct nddv1.ConditionKind) nddv1.Condition
	SetConditions(c ...nddv1.Condition)
	GetNiPoolName() string
	GetSourceTag() map[string]string
	GetSelector() map[string]string
	SetNi(string)
	HasNi() (string, bool)
}

// GetCondition of this Network Node.
func (x *Alloc) GetCondition(ct nddv1.ConditionKind) nddv1.Condition {
	return x.Status.GetCondition(ct)
}

// SetConditions of the Network Node.
func (x *Alloc) SetConditions(c ...nddv1.Condition) {
	x.Status.SetConditions(c...)
}

func (n *Alloc) GetNiPoolName() string {
	if reflect.ValueOf(n.Spec.NiPoolName).IsZero() {
		return ""
	}
	return *n.Spec.NiPoolName
}

func (n *Alloc) GetSourceTag() map[string]string {
	s := make(map[string]string)
	if reflect.ValueOf(n.Spec.Alloc.SourceTag).IsZero() {
		return s
	}
	for _, tag := range n.Spec.Alloc.SourceTag {
		s[*tag.Key] = *tag.Value
	}
	return s
}

func (n *Alloc) GetSelector() map[string]string {
	s := make(map[string]string)
	if reflect.ValueOf(n.Spec.Alloc.Selector).IsZero() {
		return s
	}
	for _, tag := range n.Spec.Alloc.Selector {
		s[*tag.Key] = *tag.Value
	}
	return s
}

func (n *Alloc) SetNi(key string) {
	n.Status = AllocStatus{
		Alloc: &NddrNiPoolAlloc{
			State: &NddrAllocState{
				NetworkInstance: &key,
			},
		},
	}
}

func (n *Alloc) HasNi() (string, bool) {
	fmt.Printf("HasNi: %#v\n", n.Status.Alloc)
	if n.Status.Alloc != nil && n.Status.Alloc.State != nil && n.Status.Alloc.State.NetworkInstance != nil {
		return *n.Status.Alloc.State.NetworkInstance, true
	}
	return "", false

}
