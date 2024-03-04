package util

import (
	"context"
	"time"

	"github.com/daytonaio/daytona/pkg/types"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func StopContainer(client *client.Client, project *types.Project) error {
	containerName := GetContainerName(project)
	ctx := context.Background()

	err := client.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		return err
	}

	//	TODO: timeout
	for {
		inspect, err := client.ContainerInspect(ctx, containerName)
		if err != nil {
			return err
		}

		if !inspect.State.Running {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}
