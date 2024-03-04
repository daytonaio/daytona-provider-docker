package util

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func RemoveNetwork(client *client.Client, workspaceId string) error {
	log.Debug("Removing network")
	ctx := context.Background()

	networks, err := client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Name == workspaceId {
			err := client.NetworkRemove(ctx, network.ID)
			if err != nil {
				return err
			}
		}
	}

	log.Debug("Network removed")

	return nil
}
