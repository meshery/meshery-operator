package cache

type Cache interface {
	Read()
	Write()
	ListNamespaces()
}
