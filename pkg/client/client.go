package client

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/daytonaio/daytona-provider-docker/pkg/ssh_tunnel/util"
	"github.com/daytonaio/daytona-provider-docker/pkg/types"

	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

func GetClient(targetOptions types.TargetOptions, sockDir string) (*client.Client, error) {
	if targetOptions.RemoteHostname == nil {
		return getLocalClient(targetOptions)
	}

	return getRemoteClient(targetOptions, sockDir)
}

func getLocalClient(targetOptions types.TargetOptions) (*client.Client, error) {
	schema := "unix://"
	if runtime.GOOS == "windows" {
		schema = "npipe://"
	}

	if targetOptions.SockPath != nil && *targetOptions.SockPath != "" && *targetOptions.SockPath != "/var/run/docker.sock" {
		cli, err := client.NewClientWithOpts(client.WithHost(fmt.Sprintf("%s%s", schema, *targetOptions.SockPath)), client.WithAPIVersionNegotiation())
		if err != nil {
			return nil, err
		}

		return cli, nil
	}

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

	remoteSockPath := "/var/run/docker.sock"
	if targetOptions.SockPath != nil && *targetOptions.SockPath != "" {
		remoteSockPath = *targetOptions.SockPath
	}

	startedChan, errChan := util.ForwardRemoteUnixSock(
		context.Background(),
		targetOptions,
		localSockPath,
		remoteSockPath,
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
