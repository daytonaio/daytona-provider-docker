package util

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func RemoveNetwork(workspaceId string) error {
	log.Debug("Removing network")
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Name == workspaceId {
			err := cli.NetworkRemove(ctx, network.ID)
			if err != nil {
				return err
			}
		}
	}

	log.Debug("Network removed")

	return nil
}
