package provider

import (
	"errors"
	"io"

	log_writers "github.com/daytonaio/daytona-provider-docker/internal/log"
	"github.com/daytonaio/daytona-provider-docker/pkg/types"

	"github.com/daytonaio/daytona/pkg/docker"
	"github.com/daytonaio/daytona/pkg/logs"
	"github.com/daytonaio/daytona/pkg/provider"
	provider_util "github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/ssh"
)

func (p DockerProvider) CreateTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		targetLogWriter := loggerFactory.CreateTargetLogger(targetReq.Target.Id, targetReq.Target.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, targetLogWriter)
		defer targetLogWriter.Close()
	}

	dockerClient, err := p.getClient(targetReq.Target.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	targetDir, err := p.getTargetDir(targetReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(targetReq.Target.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}
	if sshClient != nil {
		defer sshClient.Close()
	}

	return new(provider_util.Empty), dockerClient.CreateTarget(targetReq.Target, targetDir, logWriter, sshClient)
}

func (p DockerProvider) CreateWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	if p.DaytonaDownloadUrl == nil {
		return new(provider_util.Empty), errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		workspaceLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceReq.Workspace.Id, workspaceReq.Workspace.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, workspaceLogWriter)
		defer workspaceLogWriter.Close()
	}

	dockerClient, err := p.getClient(workspaceReq.Workspace.Target.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	workspaceDir, err := p.getWorkspaceDir(workspaceReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	_, isLocal, err := types.ParseTargetConfigOptions(workspaceReq.Workspace.Target.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	var sshClient *ssh.Client
	if !isLocal {
		sshClient, err = p.getSshClient(workspaceReq.Workspace.Target.Options)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer sshClient.Close()
	}

	return new(provider_util.Empty), dockerClient.CreateWorkspace(&docker.CreateWorkspaceOptions{
		Workspace:    workspaceReq.Workspace,
		WorkspaceDir: workspaceDir,
		Cr:           workspaceReq.ContainerRegistry,
		LogWriter:    logWriter,
		Gpc:          workspaceReq.GitProviderConfig,
		SshClient:    sshClient,
	})
}
