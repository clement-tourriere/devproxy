package docker

import (
	"context"
	"log/slog"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type Monitor struct {
	client *client.Client
	logger *slog.Logger
}

type ContainerEvent struct {
	Action    string
	Container types.ContainerJSON
}

func NewMonitor(logger *slog.Logger) (*Monitor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Monitor{
		client: cli,
		logger: logger,
	}, nil
}

func (m *Monitor) Start(ctx context.Context, eventsChan chan<- ContainerEvent) error {
	filterArgs := filters.NewArgs()
	filterArgs.Add("type", "container")
	filterArgs.Add("event", "start")
	filterArgs.Add("event", "stop")
	filterArgs.Add("event", "die")

	eventOptions := types.EventsOptions{
		Filters: filterArgs,
	}

	eventsCh, errCh := m.client.Events(ctx, eventOptions)

	go func() {
		defer close(eventsChan)

		for {
			select {
			case event := <-eventsCh:
				m.handleEvent(ctx, event, eventsChan)
			case err := <-errCh:
				if err != nil {
					m.logger.Error("Docker events error", "error", err)
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (m *Monitor) handleEvent(ctx context.Context, event events.Message, eventsChan chan<- ContainerEvent) {
	if event.Type != "container" {
		return
	}

	containerInfo, err := m.client.ContainerInspect(ctx, event.Actor.ID)
	if err != nil {
		m.logger.Error("Failed to inspect container", "container_id", event.Actor.ID, "error", err)
		return
	}

	m.logger.Info("Container event",
		"action", event.Action,
		"container_name", containerInfo.Name,
		"container_id", event.Actor.ID[:12])

	select {
	case eventsChan <- ContainerEvent{
		Action:    string(event.Action),
		Container: containerInfo,
	}:
	case <-ctx.Done():
		return
	}
}

func (m *Monitor) GetRunningContainers(ctx context.Context) ([]types.Container, error) {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{
		All: false,
	})
	if err != nil {
		return nil, err
	}

	return containers, nil
}

func (m *Monitor) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.client.ContainerInspect(ctx, containerID)
}
