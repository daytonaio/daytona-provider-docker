package util

import (
	"context"

	"github.com/daytonaio/daytona/pkg/types"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func GetContainerName(project *types.Project) string {
	return project.WorkspaceId + "-" + project.Name
}

func GetVolumeName(project *types.Project) string {
	return GetContainerName(project)
}

func GetContainerInfo(client *client.Client, project *types.Project) (*docker_types.ContainerJSON, error) {
	ctx := context.Background()

	inspect, err := client.ContainerInspect(ctx, GetContainerName(project))
	if err != nil {
		return nil, err
	}

	return &inspect, nil
}
