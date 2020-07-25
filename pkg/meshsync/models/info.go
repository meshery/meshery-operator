package models

import "sync"

// MeshInfo is the meshsync native, mesh agnostic representation of a deployed service-mesh
// and any related state information
// TODO (Nitish Malhotra) : Enhance this model
type MeshInfo struct {
	mu *sync.RWMutex
	// Mesh agnostic, abstract data to represent a service mesh
	// deployment in meshsync

	version *string

	// Opaque field is used to hold mesh specific data.
	// The conversion of this data to a known type is up to
	// the consumer of the data
	opaque interface{}
}

func NewMeshInfo() *MeshInfo {
	return &MeshInfo{
		mu: &sync.RWMutex{},
	}
}

// SetVersion of the deployed mesh
func (mi *MeshInfo) SetVersion(v *string) {
	mi.version = v
}

// SetOpaque data unique to the deployed mesh
func (mi *MeshInfo) SetOpaque(data interface{}) {
	mi.opaque = data
}
