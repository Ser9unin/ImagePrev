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
	Host := os.Getenv("HOST")
	if Host == "" {
		Host = "localhost"
	}
	Port := os.Getenv("PORT")
	if Port == "" {
		Port = ":8000"
	}
	server := SrvCfg{
		Host: Host,
		Port: Port,
	}

	cacheCapStr := os.Getenv("CACHE_CAPACITY")
	cacheCapInt, err := strconv.Atoi(cacheCapStr)
	if err != nil {
		cacheCapInt = 3
		log.Printf("can't get cache cap, set to default = %d \n", cacheCapInt)
	}

	cache := CacheCfg{
		Capacity: cacheCapInt,
	}

	return Config{
		Server: server,
		Cache:  cache,
	}
}
