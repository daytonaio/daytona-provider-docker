package util

import (
	"context"

	"github.com/daytonaio/daytona/grpc/proto/types"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func RemoveContainer(project *types.Project) error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	err = cli.ContainerRemove(ctx, GetContainerName(project), docker_types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}

	err = cli.VolumeRemove(ctx, GetVolumeName(project), true)
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}

	return nil
}
