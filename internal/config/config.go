package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DevProxy  DevProxyConfig
	Dashboard DashboardConfig
}

type DevProxyConfig struct {
	LogLevel      string
	CaddyAdminURL string
	DomainSuffix  string
}

type DashboardConfig struct {
	RefreshInterval  int
	ExcludedProjects []string
	ShowAllProjects  bool
	Addr             string
}

// Load configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		DevProxy: DevProxyConfig{
			LogLevel:      getEnv("DEVPROXY_LOG_LEVEL", "info"),
			CaddyAdminURL: getEnv("CADDY_ADMIN_URL", "http://localhost:2019"),
			DomainSuffix:  getEnv("DEVPROXY_DOMAIN_SUFFIX", "localhost"),
		},
		Dashboard: DashboardConfig{
			RefreshInterval:  getEnvInt("DEVPROXY_DASHBOARD_REFRESH", 30),
			ExcludedProjects: getEnvList("DEVPROXY_DASHBOARD_EXCLUDE", []string{"devproxy"}),
			ShowAllProjects:  getEnvBool("DEVPROXY_DASHBOARD_SHOW_ALL", false),
			Addr:             getEnv("DASHBOARD_ADDR", ":8080"),
		},
	}
}

// getEnv gets environment variable with default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as integer with default fallback
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvBool gets environment variable as boolean with default fallback
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// getEnvList gets environment variable as comma-separated list with default fallback
func getEnvList(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		list := strings.Split(value, ",")
		// Trim spaces from each item
		for i, item := range list {
			list[i] = strings.TrimSpace(item)
		}
		return list
	}
	return defaultValue
}
