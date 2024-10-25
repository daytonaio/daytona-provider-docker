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

func (p DockerProvider) CreateProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	if p.DaytonaDownloadUrl == nil {
		return new(provider_util.Empty), errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		projectLogWriter := loggerFactory.CreateProjectLogger(projectReq.Project.TargetId, projectReq.Project.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, projectLogWriter)
		defer projectLogWriter.Close()
	}

	dockerClient, err := p.getClient(projectReq.TargetConfigOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	projectDir, err := p.getProjectDir(projectReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	var sshClient *ssh.Client
	if projectReq.Project.TargetConfig != "local" {
		sshClient, err = p.getSshClient(projectReq.Project.TargetConfig, projectReq.TargetConfigOptions)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer sshClient.Close()
	}

	return new(provider_util.Empty), dockerClient.CreateProject(&docker.CreateProjectOptions{
		Project:    projectReq.Project,
		ProjectDir: projectDir,
		Cr:         projectReq.ContainerRegistry,
		LogWriter:  logWriter,
		Gpc:        projectReq.GitProviderConfig,
		SshClient:  sshClient,
	})
}
