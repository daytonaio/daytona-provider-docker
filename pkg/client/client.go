package client

import (
	"context"
	"fmt"
	"os"
	"path"
	"provider/pkg/ssh_tunnel/util"
	"provider/pkg/types"

	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

func GetClient(targetOptions types.TargetOptions, pluginPath string) (*client.Client, error) {
	if targetOptions.RemoteHostname == nil {
		return getLocalClient()
	}

	return getRemoteClient(targetOptions, pluginPath)
}

func getLocalClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func getRemoteClient(targetOptions types.TargetOptions, pluginPath string) (*client.Client, error) {
	localSockPath, err := forwardDockerSock(targetOptions, pluginPath)
	if err != nil {
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.WithHost(fmt.Sprintf("unix://%s", localSockPath)), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func forwardDockerSock(targetOptions types.TargetOptions, pluginPath string) (string, error) {
	localSockPath := path.Join(pluginPath, fmt.Sprintf("daytona-%s-docker.sock", *targetOptions.RemoteHostname))

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
