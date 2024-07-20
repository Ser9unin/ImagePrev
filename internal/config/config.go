package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Server SrvCfg
	Cache  CacheCfg
}

type SrvCfg struct {
	Host string
	Port string
}

type CacheCfg struct {
	Capacity int
}

func New() Config {
	server := SrvCfg{
		Host: os.Getenv("HOST"),
		Port: os.Getenv("PORT"),
	}

	cacheCapStr := os.Getenv("CACHE_CAPACITY")
	cacheCapInt, err := strconv.Atoi(cacheCapStr)
	if err != nil {
		log.Println("can't get cache cap, set to default = 1")
		cacheCapInt = 1
	}

	cache := CacheCfg{
		Capacity: cacheCapInt,
	}

	return Config{
		Server: server,
		Cache:  cache,
	}
}
