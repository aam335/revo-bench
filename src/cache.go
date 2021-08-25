package main

import (
	"log"
	"sync"
	"time"
)

// если нужно масштабирование - redis с настройками в Config

var cache Cache

type cacheItem struct {
	parallels int
	stt       time.Time
}

type Cache struct {
	m sync.Map
}

func (c *Cache) Get(site string, ttl time.Duration) int { // returns -1, если нет значения или протухло
	iface, ok := cache.m.Load(site)
	if !ok {
		return -1
	}
	if cached, ok := iface.(*cacheItem); ok && time.Since(cached.stt) < ttl {
		log.Printf("from cache %v: %#v", site, cached)
		return cached.parallels
	}
	return -1
}

func (c *Cache) Put(site string, parallels int) {
	v := &cacheItem{parallels: parallels, stt: time.Now()}
	log.Printf("2cache %v:%#v", site, v)
	c.m.Store(site, v)
}
