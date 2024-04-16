package util

import (
	"context"

	"github.com/daytonaio/daytona/pkg/workspace"
	docker_types "github.com/docker/docker/api/types"
	docker_client "github.com/docker/docker/client"
)

func RemoveContainer(client *docker_client.Client, project *workspace.Project) error {
	ctx := context.Background()

	err := client.ContainerRemove(ctx, GetContainerName(project), docker_types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil && !docker_client.IsErrNotFound(err) {
		return err
	}

	err = client.VolumeRemove(ctx, GetVolumeName(project), true)
	if err != nil && !docker_client.IsErrNotFound(err) {
		return err
	}

	return nil
}
