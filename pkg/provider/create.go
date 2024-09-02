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

func (p DockerProvider) CreateWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		wsLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceReq.Workspace.Id, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, wsLogWriter)
		defer wsLogWriter.Close()
	}

	dockerClient, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	workspaceDir, err := p.getWorkspaceDir(workspaceReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(workspaceReq.Workspace.Target, workspaceReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}
	if sshClient != nil {
		defer sshClient.Close()
	}

	return new(provider_util.Empty), dockerClient.CreateWorkspace(workspaceReq.Workspace, workspaceDir, logWriter, sshClient)
}

func (p DockerProvider) CreateProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	if p.DaytonaDownloadUrl == nil {
		return new(provider_util.Empty), errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		projectLogWriter := loggerFactory.CreateProjectLogger(projectReq.Project.WorkspaceId, projectReq.Project.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, projectLogWriter)
		defer projectLogWriter.Close()
	}

	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	projectDir, err := p.getProjectDir(projectReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	var sshClient *ssh.Client
	if projectReq.Project.Target != "local" {
		sshClient, err = p.getSshClient(projectReq.Project.Target, projectReq.TargetOptions)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer sshClient.Close()
	}
	// TODO: think about how to handle this since the project is cloned in dockerClient.CreateProject so we can't detect the builder type here
	// else {
	// 	builderType, err := detect.DetectProjectBuilderType(projectReq.Project.BuildConfig, projectDir, nil)
	// 	if err != nil {
	// 		return new(provider_util.Empty), err
	// 	}

	// 	if builderType != detect.BuilderTypeDevcontainer {
	// 		p.setLocalEnvOverride(projectReq.Project)
	// 	}
	// }

	return new(provider_util.Empty), dockerClient.CreateProject(&docker.CreateProjectOptions{
		Project:    projectReq.Project,
		ProjectDir: projectDir,
		Cr:         projectReq.ContainerRegistry,
		LogWriter:  logWriter,
		Gpc:        projectReq.GitProviderConfig,
		SshClient:  sshClient,
	})
}
