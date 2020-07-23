package models

// MeshInfo is the meshsync native, mesh agnostic representation of a deployed service-mesh
// and any related state information
// TODO (Nitish Malhotra) : Enhance this model
type MeshInfo struct {
	Version *string
}
