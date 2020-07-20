package models

type Info interface {
	Version() *string
	Details() MeshInfo
	Error() error
}
