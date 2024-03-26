package util

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func CreateNetwork(client *client.Client, workspaceId string, logWriter *io.Writer) error {
	if logWriter != nil {
		(*logWriter).Write([]byte("Initializing network\n"))
	}
	ctx := context.Background()

	networks, err := client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Name == workspaceId {
			if logWriter != nil {
				(*logWriter).Write([]byte("Network already exists\n"))
			}
			return nil
		}
	}

	_, err = client.NetworkCreate(ctx, workspaceId, types.NetworkCreate{
		Attachable: true,
	})
	if err != nil {
		return err
	}

	if logWriter != nil {
		(*logWriter).Write([]byte("Network initialized\n"))
	}
	return nil
}
