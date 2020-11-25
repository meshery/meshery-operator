package cache

import (
	"time"

	"github.com/allegro/bigcache"
)

type bigCache struct {
	handler *bigcache.BigCache
}

func New(minutes time.Duration) (*bigCache, error) {

	cache, _ := bigcache.NewBigCache(bigcache.DefaultConfig(minutes * time.Minute))

	return &bigCache{
		handler: cache,
	}, nil
}

func (b *bigCache) Read(result *Resources) error {
	return nil
}

func (b *bigCache) Write(result *Resources) error {
	return nil
}
