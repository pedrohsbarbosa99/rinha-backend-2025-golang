package service

import "os"

type Env struct {
	REDIS_URL              string
	PROCESSOR_DEFAULT_URL  string
	PROCESSOR_FALLBACK_URL string
}

type Config struct{}

func (Config) LoadEnv() (c Env) {
	return Env{
		REDIS_URL:              os.Getenv("REDIS_URL"),
		PROCESSOR_DEFAULT_URL:  os.Getenv("PRCESSOR_DEFAULT_URL"),
		PROCESSOR_FALLBACK_URL: os.Getenv("PROCESSOR_FALLBACK_URL"),
	}
}
