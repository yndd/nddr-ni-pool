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
	"github.com/yndd/ndd-runtime/pkg/resource"
	"github.com/yndd/ndd-runtime/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ NpList = &NiPoolList{}

// +k8s:deepcopy-gen=false
type NpList interface {
	client.ObjectList

	GetNiPools() []Np
}

func (x *NiPoolList) GetNiPools() []Np {
	xs := make([]Np, len(x.Items))
	for i, r := range x.Items {
		r := r // Pin range variable so we can take its address.
		xs[i] = &r
	}
	return xs
}

var _ Np = &NiPool{}

// +k8s:deepcopy-gen=false
type Np interface {
	resource.Object
	resource.Conditioned

	GetAllocationStrategy() string
	GetSize() uint32
	GetAllocations() int
	GetAllocatedNis() []*string
	InitializeResource() error
	SetNi(string)
	UnSetNi(string)
	UpdateNi([]*string)
	FindNi(string) bool
}

// GetCondition of this Network Node.
func (x *NiPool) GetCondition(ct nddv1.ConditionKind) nddv1.Condition {
	return x.Status.GetCondition(ct)
}

// SetConditions of the Network Node.
func (x *NiPool) SetConditions(c ...nddv1.Condition) {
	x.Status.SetConditions(c...)
}

func (n *NiPool) GetAllocationStrategy() string {
	if reflect.ValueOf(n.Spec.NipoolNiPool.AllocationStrategy).IsZero() {
		return ""
	}
	return *n.Spec.NipoolNiPool.AllocationStrategy
}

func (n *NiPool) GetSize() uint32 {
	if reflect.ValueOf(n.Spec.NipoolNiPool.Size).IsZero() {
		return 0
	}
	return *n.Spec.NipoolNiPool.Size
}

func (n *NiPool) GetAllocations() int {
	if n.Status.NipoolNiPool != nil && n.Status.NipoolNiPool.State != nil {
		return *n.Status.NipoolNiPool.State.Allocated
	}
	return 0
}

func (n *NiPool) GetAllocatedNis() []*string {
	return n.Status.NipoolNiPool.State.Used
}

func (n *NiPool) InitializeResource() error {

	// check if the pool was already initialized
	if n.Status.NipoolNiPool != nil && n.Status.NipoolNiPool.State != nil {
		// pool was already initialiazed
		return nil
	}
	size := int(*n.Spec.NipoolNiPool.Size)

	n.Status.NipoolNiPool = &NddrNiPoolNiPool{
		Size:        n.Spec.NipoolNiPool.Size,
		AdminState:  n.Spec.NipoolNiPool.AdminState,
		Description: n.Spec.NipoolNiPool.Description,
		State: &NddrNiPoolNiPoolState{
			Total:     utils.IntPtr(size),
			Allocated: utils.IntPtr(0),
			Available: utils.IntPtr(size),
			Used:      make([]*string, 0),
		},
	}
	return nil

}

func (n *NiPool) SetNi(ni string) {
	*n.Status.NipoolNiPool.State.Allocated++
	*n.Status.NipoolNiPool.State.Available--

	n.Status.NipoolNiPool.State.Used = append(n.Status.NipoolNiPool.State.Used, &ni)
}

func (n *NiPool) UnSetNi(ni string) {
	*n.Status.NipoolNiPool.State.Allocated--
	*n.Status.NipoolNiPool.State.Available++

	found := false
	idx := 0
	for i, pni := range n.Status.NipoolNiPool.State.Used {
		if *pni == ni {
			found = true
			idx = i
			break
		}
	}
	if found {
		n.Status.NipoolNiPool.State.Used = append(n.Status.NipoolNiPool.State.Used[:idx], n.Status.NipoolNiPool.State.Used[idx+1:]...)
	}
}

func (n *NiPool) UpdateNi(allocatedNis []*string) {
	*n.Status.NipoolNiPool.State.Available = *n.Status.NipoolNiPool.State.Total - len(allocatedNis)
	*n.Status.NipoolNiPool.State.Allocated = len(allocatedNis)
	n.Status.NipoolNiPool.State.Used = allocatedNis
}

func (n *NiPool) FindNi(ni string) bool {
	for _, uni := range n.Status.NipoolNiPool.State.Used {
		if *uni == ni {
			return true
		}
	}
	return false
}
