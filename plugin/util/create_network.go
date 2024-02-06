package util

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func CreateNetwork(workspaceId string) error {
	log.Debug("Initializing network")
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
			log.WithFields(log.Fields{
				"workspace": workspaceId,
			}).Debug("Network already exists")
			return nil
		}
	}

	_, err = cli.NetworkCreate(ctx, workspaceId, types.NetworkCreate{
		Attachable: true,
	})
	if err != nil {
		return err
	}

	log.Debug("Network initialized")
	return nil
}
