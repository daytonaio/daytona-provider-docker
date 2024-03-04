package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"provider/pkg/ssh_tunnel/util"
	"provider/pkg/types"

	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

func GetClient(target provider.ProviderTarget, pluginPath string) (*client.Client, error) {
	if target.Name == "local" {
		return getLocalClient()
	}

	return getRemoteClient(target, pluginPath)
}

func getLocalClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func getRemoteClient(target provider.ProviderTarget, pluginPath string) (*client.Client, error) {
	localSockPath, err := forwardDockerSock(target, pluginPath)
	if err != nil {
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.WithHost(fmt.Sprintf("unix://%s", localSockPath)), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func forwardDockerSock(target provider.ProviderTarget, pluginPath string) (string, error) {
	var targetOptions types.TargetOptions
	err := json.Unmarshal([]byte(target.Options), &targetOptions)
	if err != nil {
		return "", err
	}

	localSockPath := path.Join(pluginPath, fmt.Sprintf("daytona-%s-docker.sock", target.Name))

	if _, err := os.Stat(localSockPath); err == nil {
		return localSockPath, nil
	}

	startedChan, errChan := util.ForwardRemoteUnixSock(
		context.Background(),
		targetOptions,
		localSockPath,
		"/var/run/docker.sock",
	)

	go func() {
		err := <-errChan
		if err != nil {
			log.Error(err)
		}
		os.Remove(localSockPath)
	}()

	<-startedChan

	return localSockPath, nil
}
