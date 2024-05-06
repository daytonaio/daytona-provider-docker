package provider

import (
	"errors"
	"io"
	"os"
	"path"
	"runtime"

	internal "github.com/daytonaio/daytona-provider-docker/internal"
	log_writers "github.com/daytonaio/daytona-provider-docker/internal/log"
	"github.com/daytonaio/daytona-provider-docker/pkg/client"
	provider_types "github.com/daytonaio/daytona-provider-docker/pkg/types"

	"github.com/daytonaio/daytona/pkg/docker"
	"github.com/daytonaio/daytona/pkg/logger"
	"github.com/daytonaio/daytona/pkg/provider"
	provider_util "github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/workspace"
)

type DockerProvider struct {
	BasePath          *string
	ServerDownloadUrl *string
	ServerVersion     *string
	ServerUrl         *string
	ServerApiUrl      *string
	LogsDir           *string
	RemoteSockDir     string
}

func (p *DockerProvider) Initialize(req provider.InitializeProviderRequest) (*provider_util.Empty, error) {
	tmpDir := "/tmp"
	if runtime.GOOS == "windows" {
		tmpDir = os.TempDir()
		if tmpDir == "" {
			return new(provider_util.Empty), errors.New("could not determine temp dir")
		}
	}

	p.RemoteSockDir = path.Join(tmpDir, "target-socks")

	// Clear old sockets
	err := os.RemoveAll(p.RemoteSockDir)
	if err != nil {
		return new(provider_util.Empty), err
	}
	err = os.MkdirAll(p.RemoteSockDir, 0755)
	if err != nil {
		return new(provider_util.Empty), err
	}

	p.BasePath = &req.BasePath
	p.ServerDownloadUrl = &req.ServerDownloadUrl
	p.ServerVersion = &req.ServerVersion
	p.ServerUrl = &req.ServerUrl
	p.ServerApiUrl = &req.ServerApiUrl
	p.LogsDir = &req.LogsDir

	return new(provider_util.Empty), nil
}

func (p DockerProvider) GetInfo() (provider.ProviderInfo, error) {
	return provider.ProviderInfo{
		Name:    "docker-provider",
		Version: internal.Version,
	}, nil
}

func (p DockerProvider) GetTargetManifest() (*provider.ProviderTargetManifest, error) {
	return provider_types.GetTargetManifest(), nil
}

func (p DockerProvider) GetDefaultTargets() (*[]provider.ProviderTarget, error) {
	info, err := p.GetInfo()
	if err != nil {
		return nil, err
	}

	defaultTargets := []provider.ProviderTarget{
		{
			Name:         "local",
			ProviderInfo: info,
			Options:      "{\n\t\"Sock Path\": \"/var/run/docker.sock\"\n}",
		},
	}
	return &defaultTargets, nil
}

func (p DockerProvider) CreateWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logger.NewLoggerFactory(*p.LogsDir)
		wsLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceReq.Workspace.Id)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, wsLogWriter)
		defer wsLogWriter.Close()
	}

	dockerClient, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), dockerClient.CreateWorkspace(workspaceReq.Workspace, logWriter)
}

func (p DockerProvider) StartWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	return new(provider_util.Empty), nil
}

func (p DockerProvider) StopWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	return new(provider_util.Empty), nil
}

func (p DockerProvider) DestroyWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), dockerClient.DestroyWorkspace(workspaceReq.Workspace)
}

func (p DockerProvider) GetWorkspaceInfo(workspaceReq *provider.WorkspaceRequest) (*workspace.WorkspaceInfo, error) {
	dockerClient, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return nil, err
	}

	return dockerClient.GetWorkspaceInfo(workspaceReq.Workspace)
}

func (p DockerProvider) CreateProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	if p.ServerDownloadUrl == nil {
		return new(provider_util.Empty), errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logger.NewLoggerFactory(*p.LogsDir)
		projectLogWriter := loggerFactory.CreateProjectLogger(projectReq.Project.WorkspaceId, projectReq.Project.Name)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, projectLogWriter)
		defer projectLogWriter.Close()
	}

	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	err = dockerClient.CreateProject(projectReq.Project, *p.ServerDownloadUrl, projectReq.ContainerRegistry, logWriter)
	if err != nil {
		return new(provider_util.Empty), err
	}

	err = dockerClient.StartProject(projectReq.Project)
	if err != nil {
		return new(provider_util.Empty), err
	}

	go func() {
		err = dockerClient.GetContainerLogs(dockerClient.GetProjectContainerName(projectReq.Project), logWriter)
		if err != nil {
			logWriter.Write([]byte(err.Error()))
		}
	}()

	return new(provider_util.Empty), nil
}

func (p DockerProvider) StartProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), dockerClient.StartProject(projectReq.Project)
}

func (p DockerProvider) StopProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), dockerClient.StopProject(projectReq.Project)
}

func (p DockerProvider) DestroyProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), dockerClient.DestroyProject(projectReq.Project)
}

func (p DockerProvider) GetProjectInfo(projectReq *provider.ProjectRequest) (*workspace.ProjectInfo, error) {
	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return nil, err
	}

	return dockerClient.GetProjectInfo(projectReq.Project)
}

func (p DockerProvider) getClient(targetOptionsJson string) (docker.IDockerClient, error) {
	targetOptions, err := provider_types.ParseTargetOptions(targetOptionsJson)
	if err != nil {
		return nil, err
	}

	client, err := client.GetClient(*targetOptions, p.RemoteSockDir)
	if err != nil {
		return nil, err
	}

	return docker.NewDockerClient(docker.DockerClientConfig{
		ApiClient: client,
	}), nil
}
