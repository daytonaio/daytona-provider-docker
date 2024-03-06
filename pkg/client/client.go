package client

import (
	"context"
	"fmt"
	"os"
	"path"
	"provider/pkg/ssh_tunnel/util"
	"provider/pkg/types"
	"strings"

	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

func GetClient(targetOptions types.TargetOptions, sockDir string) (*client.Client, error) {
	if targetOptions.RemoteHostname == nil {
		return getLocalClient()
	}

	return getRemoteClient(targetOptions, sockDir)
}

func getLocalClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func getRemoteClient(targetOptions types.TargetOptions, sockDir string) (*client.Client, error) {
	localSockPath, err := forwardDockerSock(targetOptions, sockDir)
	if err != nil {
		return nil, err
	}

	cli, err := client.NewClientWithOpts(client.WithHost(fmt.Sprintf("unix://%s", localSockPath)), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func forwardDockerSock(targetOptions types.TargetOptions, sockDir string) (string, error) {
	localSockPath := path.Join(sockDir, fmt.Sprintf("daytona-%s-docker.sock", strings.ReplaceAll(*targetOptions.RemoteHostname, ".", "-")))

	if _, err := os.Stat(path.Dir(localSockPath)); err != nil {
		err := os.MkdirAll(path.Dir(localSockPath), 0755)
		if err != nil {
			return "", err
		}
	}

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
			startedChan <- false
			os.Remove(localSockPath)
		}
	}()

	<-startedChan

	return localSockPath, nil
}
