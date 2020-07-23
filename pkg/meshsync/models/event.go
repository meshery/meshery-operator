package models

import "fmt"

// EventType identifies the type of event being processed
type EventType int

const (
	// Discovery event is the first stage to finding a deployed mesh
	Discovery EventType = iota
	// All other types are related to multi-stage fingerprinting
	// TODO (Nitish Malhotra) : Fill me
)

// Event represents a mesh agnostic event interface
type Event interface {
	// Stringer returns a long form description/details of the mesh
	fmt.Stringer
	Details() MeshInfo
	Type() EventType
}
