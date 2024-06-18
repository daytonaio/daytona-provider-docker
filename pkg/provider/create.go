package provider

import (
	"errors"
	"fmt"
	"io"
	"os"

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
		loggerFactory := logs.NewLoggerFactory(*p.LogsDir)
		wsLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceReq.Workspace.Id, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, wsLogWriter)
		defer wsLogWriter.Close()
	}

	dockerClient, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), dockerClient.CreateWorkspace(workspaceReq.Workspace, logWriter)
}

func (p DockerProvider) CreateProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	if p.DaytonaDownloadUrl == nil {
		return new(provider_util.Empty), errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(*p.LogsDir)
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

	var sshSessionConfig *ssh.SessionConfig
	if projectReq.Project.Target == "local" {
		if projectReq.Project.Build == nil {
			p.setLocalEnvOverride(projectReq.Project)
		}
		err = os.MkdirAll(projectDir, 0755)
		if err != nil {
			return new(provider_util.Empty), err
		}
	} else {
		sshSessionConfig, err = p.getSshSessionConfig(projectReq.TargetOptions)
		if err != nil {
			return new(provider_util.Empty), err
		}

		client, err := ssh.NewClient(sshSessionConfig)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer client.Close()
		err = client.Exec(fmt.Sprintf("mkdir -p %s", projectDir), nil)
		if err != nil {
			return new(provider_util.Empty), err
		}
	}

	return new(provider_util.Empty), dockerClient.CreateProject(&docker.CreateProjectOptions{
		Project:          projectReq.Project,
		ProjectDir:       projectDir,
		Cr:               projectReq.ContainerRegistry,
		LogWriter:        logWriter,
		Gpc:              projectReq.GitProviderConfig,
		SshSessionConfig: sshSessionConfig,
	})
}
