package config

import (
	"os"
	"strconv"
)

type Config struct {
    OpenAIAPIKey    string
    OpenAIBaseURL   string
    Model           string
    MaxTokens       int
    MemoryCapacity  int
    Verbose         bool
    MaxConcurrency  int
    RequestTimeout  int
    ReasoningEnabled bool
    ReasoningMaxSteps int
}

func Load() *Config {
    cfg := &Config{
        OpenAIAPIKey:   os.Getenv("OPENAI_API_KEY"),
        OpenAIBaseURL:  os.Getenv("OPENAI_API_BASE_URL"),
        Model:          getEnvWithDefault("MODEL", "gpt-4"),
        MaxTokens:      getEnvIntWithDefault("MAX_TOKENS", 1024),
        MemoryCapacity: getEnvIntWithDefault("MEMORY_CAPACITY", 40),
        Verbose:        getEnvBoolWithDefault("VERBOSE", false),
        MaxConcurrency: getEnvIntWithDefault("MAX_CONCURRENCY", 5),
        RequestTimeout: getEnvIntWithDefault("REQUEST_TIMEOUT", 30),
        ReasoningEnabled: getEnvBoolWithDefault("REASONING_ENABLED", false),
        ReasoningMaxSteps: getEnvIntWithDefault("REASONING_MAX_STEPS", 10),
    }

    return cfg
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
