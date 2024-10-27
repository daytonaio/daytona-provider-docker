package provider

import (
	"errors"
	"io"

	log_writers "github.com/daytonaio/daytona-provider-docker/internal/log"

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
		targetLogWriter := loggerFactory.CreateTargetLogger(targetReq.Target.Id, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, targetLogWriter)
		defer targetLogWriter.Close()
	}

	dockerClient, err := p.getClient(targetReq.TargetConfigOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	targetDir, err := p.getTargetDir(targetReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(targetReq.Target.TargetConfig, targetReq.TargetConfigOptions)
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
		workspaceLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceReq.Workspace.TargetId, workspaceReq.Workspace.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, workspaceLogWriter)
		defer workspaceLogWriter.Close()
	}

	dockerClient, err := p.getClient(workspaceReq.TargetConfigOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	workspaceDir, err := p.getWorkspaceDir(workspaceReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	var sshClient *ssh.Client
	if workspaceReq.Workspace.TargetConfig != "local" {
		sshClient, err = p.getSshClient(workspaceReq.Workspace.TargetConfig, workspaceReq.TargetConfigOptions)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer sshClient.Close()
	}

	return new(provider_util.Empty), dockerClient.CreateWorkspace(&docker.CreateWorkspaceOptions{
		Workspace:                workspaceReq.Workspace,
		WorkspaceDir:             workspaceDir,
		ContainerRegistry:        workspaceReq.ContainerRegistry,
		BuilderImage:             workspaceReq.BuilderImage,
		BuilderContainerRegistry: workspaceReq.BuilderContainerRegistry,
		LogWriter:                logWriter,
		Gpc:                      workspaceReq.GitProviderConfig,
		SshClient:                sshClient,
	})
}
