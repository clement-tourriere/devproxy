package proxy

import (
	"context"
	"log/slog"
	"sync"

	"devproxy/internal/caddy"
	"devproxy/internal/config"
	"devproxy/internal/docker"

	"github.com/docker/docker/api/types"
)

type Manager struct {
	dockerMonitor   *docker.Monitor
	discovery       *docker.Discovery
	configGenerator *caddy.ConfigGenerator
	caddyClient     *caddy.Client
	logger          *slog.Logger

	mu             sync.RWMutex
	proxyTargets   map[string][]docker.ProxyTarget // container ID -> targets
	lastConfigHash string
}

func NewManager(cfg *config.Config, logger *slog.Logger) (*Manager, error) {
	monitor, err := docker.NewMonitor(logger)
	if err != nil {
		return nil, err
	}

	caddyClient := caddy.NewClient(cfg.DevProxy.CaddyAdminURL, logger)

	return &Manager{
		dockerMonitor:   monitor,
		discovery:       docker.NewDiscovery(),
		configGenerator: caddy.NewConfigGenerator(),
		caddyClient:     caddyClient,
		logger:          logger,
		proxyTargets:    make(map[string][]docker.ProxyTarget),
	}, nil
}

func (m *Manager) Start(ctx context.Context) error {
	// Wait for Caddy to be ready
	m.logger.Info("Waiting for Caddy to be ready...")
	if err := m.caddyClient.WaitForReady(ctx, 30); err != nil {
		return err
	}

	// Initialize with existing containers
	if err := m.syncExistingContainers(ctx); err != nil {
		m.logger.Error("Failed to sync existing containers", "error", err)
		return err
	}

	// Start monitoring Docker events
	eventsChan := make(chan docker.ContainerEvent, 10)
	if err := m.dockerMonitor.Start(ctx, eventsChan); err != nil {
		return err
	}

	m.logger.Info("DevProxy manager started, monitoring Docker containers...")

	// Process events
	for {
		select {
		case event := <-eventsChan:
			m.handleContainerEvent(ctx, event)
		case <-ctx.Done():
			m.logger.Info("DevProxy manager stopping...")
			return nil
		}
	}
}

func (m *Manager) syncExistingContainers(ctx context.Context) error {
	containers, err := m.dockerMonitor.GetRunningContainers(ctx)
	if err != nil {
		return err
	}

	m.logger.Info("Syncing existing containers", "count", len(containers))

	for _, container := range containers {
		containerInfo, err := m.dockerMonitor.InspectContainer(ctx, container.ID)
		if err != nil {
			m.logger.Warn("Failed to inspect container", "container_id", container.ID, "error", err)
			continue
		}

		m.addContainer(ctx, containerInfo)
	}

	return m.updateCaddyConfig(ctx)
}

func (m *Manager) handleContainerEvent(ctx context.Context, event docker.ContainerEvent) {
	switch event.Action {
	case "start":
		m.addContainer(ctx, event.Container)
	case "stop", "die":
		m.removeContainer(ctx, event.Container)
	default:
		return
	}

	if err := m.updateCaddyConfig(ctx); err != nil {
		m.logger.Error("Failed to update Caddy config", "error", err)
	}
}

func (m *Manager) addContainer(ctx context.Context, container types.ContainerJSON) {
	targets := m.discovery.ExtractProxyTargets(container)
	if len(targets) == 0 {
		return
	}

	containerKey := m.discovery.GetContainerKey(container)

	m.mu.Lock()
	m.proxyTargets[containerKey] = targets
	m.mu.Unlock()

	for _, target := range targets {
		m.logger.Info("Added proxy target",
			"domain", target.Domain,
			"container_ip", target.ContainerIP,
			"port", target.Port,
			"container", container.Name)
	}
}

func (m *Manager) removeContainer(ctx context.Context, container types.ContainerJSON) {
	containerKey := m.discovery.GetContainerKey(container)

	m.mu.Lock()
	targets, exists := m.proxyTargets[containerKey]
	if exists {
		delete(m.proxyTargets, containerKey)
	}
	m.mu.Unlock()

	if exists {
		for _, target := range targets {
			m.logger.Info("Removed proxy target",
				"domain", target.Domain,
				"container", container.Name)
		}
	}
}

func (m *Manager) updateCaddyConfig(ctx context.Context) error {
	m.mu.RLock()
	var allTargets []docker.ProxyTarget
	for _, targets := range m.proxyTargets {
		allTargets = append(allTargets, targets...)
	}
	m.mu.RUnlock()

	config, err := m.configGenerator.GenerateConfig(allTargets)
	if err != nil {
		return err
	}

	configBytes, err := m.configGenerator.SerializeConfig(config)
	if err != nil {
		return err
	}

	// Check if config has changed
	configHash := m.hashConfig(configBytes)
	if configHash == m.lastConfigHash {
		return nil
	}

	if err := m.caddyClient.UpdateConfig(ctx, config); err != nil {
		return err
	}

	m.lastConfigHash = configHash
	m.logger.Info("Updated Caddy configuration", "proxy_targets", len(allTargets))

	return nil
}

func (m *Manager) hashConfig(config []byte) string {
	// Simple hash for change detection
	return string(config)
}

func (m *Manager) GetProxyTargets() map[string][]docker.ProxyTarget {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]docker.ProxyTarget)
	for k, v := range m.proxyTargets {
		result[k] = v
	}
	return result
}

func (m *Manager) GetRunningContainers(ctx context.Context) ([]types.Container, error) {
	return m.dockerMonitor.GetRunningContainers(ctx)
}

func (m *Manager) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.dockerMonitor.InspectContainer(ctx, containerID)
}

func (m *Manager) GetDiscovery() *docker.Discovery {
	return m.discovery
}
