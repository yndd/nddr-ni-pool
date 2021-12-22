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

// NddrNiPool struct
type NddrNiPool struct {
	NiPool []*NddrNiPoolNiPool `json:"ni-pool,omitempty"`
}

// NddrNiPoolNiPool struct
type NddrNiPoolNiPool struct {
	AdminState         *string                `json:"admin-state,omitempty"`
	AllocationStrategy *string                `json:"allocation-strategy,omitempty"`
	Size               *uint32                `json:"size,omitempty"`
	Description        *string                `json:"description,omitempty"`
	Name               *string                `json:"name,omitempty"`
	State              *NddrNiPoolNiPoolState `json:"state,omitempty"`
}

// NddrNiPoolNiPoolState struct
type NddrNiPoolNiPoolState struct {
	Total     *int      `json:"total,omitempty"`
	Allocated *int      `json:"allocated,omitempty"`
	Available *int      `json:"available,omitempty"`
	Used      []*string `json:"used,omitempty"`
}

// Root is the root of the schema
type Root struct {
	NipoolNddrNiPool *NddrNiPool `json:"nddr-ni-pool,omitempty"`
}
