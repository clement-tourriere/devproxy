package docker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
)

type ProxyTarget struct {
	Domain      string
	ContainerIP string
	Port        int
	IsSecure    bool
}

type Discovery struct{}

func NewDiscovery() *Discovery {
	return &Discovery{}
}

func (d *Discovery) ExtractProxyTargets(container types.ContainerJSON) []ProxyTarget {
	var targets []ProxyTarget

	if !d.shouldProxy(container) {
		return targets
	}

	domains := d.extractDomains(container)
	containerIP := d.extractContainerIP(container)
	port := d.extractPort(container)

	if port == 0 || containerIP == "" {
		return targets
	}

	for _, domain := range domains {
		targets = append(targets, ProxyTarget{
			Domain:      domain,
			ContainerIP: containerIP,
			Port:        port,
			IsSecure:    true, // Always use HTTPS
		})
	}

	return targets
}

func (d *Discovery) shouldProxy(container types.ContainerJSON) bool {
	// Skip if container is not running
	if !container.State.Running {
		return false
	}

	// Skip if explicitly disabled
	if val, exists := container.Config.Labels["devproxy.enabled"]; exists && val == "false" {
		return false
	}

	// Always allow containers with explicit domain labels
	if _, hasCustomDomain := container.Config.Labels["devproxy.domain"]; hasCustomDomain {
		return true
	}

	// Skip devproxy and caddy containers (unless they have custom domain)
	name := strings.TrimPrefix(container.Name, "/")
	if strings.HasPrefix(name, "devproxy") || strings.HasPrefix(name, "caddy") {
		return false
	}

	return true
}

func (d *Discovery) extractDomains(container types.ContainerJSON) []string {
	var domains []string

	// Check for custom domain in labels (highest priority)
	if customDomain, exists := container.Config.Labels["devproxy.domain"]; exists {
		domains = append(domains, customDomain)
		return domains
	}

	containerName := strings.TrimPrefix(container.Name, "/")

	// Check if it's part of a compose project
	if projectName, exists := container.Config.Labels["com.docker.compose.project"]; exists {
		serviceName, serviceExists := container.Config.Labels["com.docker.compose.service"]
		if serviceExists {
			// For compose services: service.project.localhost
			domains = append(domains, fmt.Sprintf("%s.%s.localhost", serviceName, projectName))
			return domains
		}
	}

	// For standalone containers: container_name.localhost
	domains = append(domains, fmt.Sprintf("%s.localhost", containerName))

	return domains
}

func (d *Discovery) extractContainerIP(container types.ContainerJSON) string {
	// First try to get IP from custom networks
	for networkName, network := range container.NetworkSettings.Networks {
		if networkName != "bridge" && network.IPAddress != "" {
			return network.IPAddress
		}
	}

	// Fallback to default bridge network
	if container.NetworkSettings.DefaultNetworkSettings.IPAddress != "" {
		return container.NetworkSettings.DefaultNetworkSettings.IPAddress
	}

	return ""
}

func (d *Discovery) extractPort(container types.ContainerJSON) int {
	// Check for custom port in labels
	if customPort, exists := container.Config.Labels["devproxy.port"]; exists {
		if port, err := strconv.Atoi(customPort); err == nil {
			return port
		}
	}

	// Check for custom port in environment variables
	for _, env := range container.Config.Env {
		if strings.HasPrefix(env, "DEVPROXY_PORT=") {
			if port, err := strconv.Atoi(strings.TrimPrefix(env, "DEVPROXY_PORT=")); err == nil {
				return port
			}
		}
	}

	// Get the first exposed port
	for portStr := range container.Config.ExposedPorts {
		port, err := nat.ParsePort(portStr.Port())
		if err == nil {
			return port
		}
	}

	// If no exposed ports, try common web ports
	commonPorts := []int{80, 8080, 3000, 8000, 5000}
	for _, port := range commonPorts {
		// This is a heuristic - in a real implementation you might want to
		// actually check if the port is listening
		return port
	}

	return 0
}

func (d *Discovery) GetContainerKey(container types.ContainerJSON) string {
	return container.ID
}
