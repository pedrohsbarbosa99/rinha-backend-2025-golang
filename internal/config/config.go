package config

import "os"

var REDIS_URL = os.Getenv("REDIS_URL")
var PROCESSOR_DEFAULT_URL = os.Getenv("PROCESSOR_DEFAULT_URL")
var PROCESSOR_FALLBACK_URL = os.Getenv("PROCESSOR_FALLBACK_URL")
