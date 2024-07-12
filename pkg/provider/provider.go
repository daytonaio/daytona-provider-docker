package provider

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"

	internal "github.com/daytonaio/daytona-provider-docker/internal"
	log_writers "github.com/daytonaio/daytona-provider-docker/internal/log"
	"github.com/daytonaio/daytona-provider-docker/pkg/client"
	provider_types "github.com/daytonaio/daytona-provider-docker/pkg/types"

	"github.com/daytonaio/daytona/pkg/docker"
	"github.com/daytonaio/daytona/pkg/logs"
	"github.com/daytonaio/daytona/pkg/provider"
	provider_util "github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/ssh"
	"github.com/daytonaio/daytona/pkg/workspace"
)

type DockerProvider struct {
	BasePath           *string
	DaytonaDownloadUrl *string
	DaytonaVersion     *string
	ServerUrl          *string
	ApiUrl             *string
	LogsDir            *string
	ApiPort            *uint32
	ServerPort         *uint32
	RemoteSockDir      string
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
	p.DaytonaDownloadUrl = &req.DaytonaDownloadUrl
	p.DaytonaVersion = &req.DaytonaVersion
	p.ServerUrl = &req.ServerUrl
	p.ApiUrl = &req.ApiUrl
	p.LogsDir = &req.LogsDir
	p.ApiPort = &req.ApiPort
	p.ServerPort = &req.ServerPort

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

	err = dockerClient.DestroyWorkspace(workspaceReq.Workspace, workspaceDir, sshClient)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), nil
}

func (p DockerProvider) GetWorkspaceInfo(workspaceReq *provider.WorkspaceRequest) (*workspace.WorkspaceInfo, error) {
	dockerClient, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return nil, err
	}

	return dockerClient.GetWorkspaceInfo(workspaceReq.Workspace)
}

func (p DockerProvider) StartProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	projectDir, err := p.getProjectDir(projectReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(*p.LogsDir)
		projectLogWriter := loggerFactory.CreateProjectLogger(projectReq.Project.WorkspaceId, projectReq.Project.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, projectLogWriter)
		defer projectLogWriter.Close()
	}

	downloadUrl := *p.DaytonaDownloadUrl
	var sshClient *ssh.Client

	if projectReq.Project.Target == "local" {
		if projectReq.Project.Build == nil {
			parsed, err := url.Parse(downloadUrl)
			if err != nil {
				return new(provider_util.Empty), err
			}
			parsed.Host = fmt.Sprintf("host.docker.internal:%d", *p.ApiPort)
			parsed.Scheme = "http"
			downloadUrl = parsed.String()
		}
	} else {
		sshClient, err = p.getSshClient(projectReq.Project.Target, projectReq.TargetOptions)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer sshClient.Close()
	}

	err = dockerClient.StartProject(&docker.CreateProjectOptions{
		Project:    projectReq.Project,
		ProjectDir: projectDir,
		Cr:         projectReq.ContainerRegistry,
		LogWriter:  logWriter,
		Gpc:        projectReq.GitProviderConfig,
		SshClient:  sshClient,
	}, downloadUrl)
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

func (p DockerProvider) StopProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(*p.LogsDir)
		projectLogWriter := loggerFactory.CreateProjectLogger(projectReq.Project.WorkspaceId, projectReq.Project.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, projectLogWriter)
		defer projectLogWriter.Close()
	}

	return new(provider_util.Empty), dockerClient.StopProject(projectReq.Project, logWriter)
}

func (p DockerProvider) DestroyProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	projectDir, err := p.getProjectDir(projectReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(projectReq.Project.Target, projectReq.TargetOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}
	if sshClient != nil {
		defer sshClient.Close()
	}

	err = dockerClient.DestroyProject(projectReq.Project, projectDir, sshClient)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), nil
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

// If the project is running locally, we override the env vars to use the host.docker.internal address
func (p DockerProvider) setLocalEnvOverride(project *workspace.Project) {
	project.EnvVars["DAYTONA_SERVER_URL"] = fmt.Sprintf("http://host.docker.internal:%d", *p.ServerPort)
	project.EnvVars["DAYTONA_SERVER_API_URL"] = fmt.Sprintf("http://host.docker.internal:%d", *p.ApiPort)
}

func (p *DockerProvider) getProjectDir(projectReq *provider.ProjectRequest) (string, error) {
	if projectReq.Project.Target == "local" {
		return filepath.Join(*p.BasePath, projectReq.Project.WorkspaceId, fmt.Sprintf("%s-%s", projectReq.Project.WorkspaceId, projectReq.Project.Name)), nil
	}

	targetOptions, err := provider_types.ParseTargetOptions(projectReq.TargetOptions)
	if err != nil {
		return "", err
	}

	// Using path instead of filepath because we always want to use / as the separator
	return path.Join(*targetOptions.WorkspaceDataDir, projectReq.Project.WorkspaceId, fmt.Sprintf("%s-%s", projectReq.Project.WorkspaceId, projectReq.Project.Name)), nil
}

func (p *DockerProvider) getWorkspaceDir(workspaceReq *provider.WorkspaceRequest) (string, error) {
	if workspaceReq.Workspace.Target == "local" {
		return filepath.Join(*p.BasePath, workspaceReq.Workspace.Id), nil
	}

	targetOptions, err := provider_types.ParseTargetOptions(workspaceReq.TargetOptions)
	if err != nil {
		return "", err
	}

	// Using path instead of filepath because we always want to use / as the separator
	return path.Join(*targetOptions.WorkspaceDataDir, workspaceReq.Workspace.Id), nil
}

func (p *DockerProvider) getSshClient(targetName string, targetOptionsJson string) (*ssh.Client, error) {
	if targetName == "local" {
		return nil, nil
	}

	targetOptions, err := provider_types.ParseTargetOptions(targetOptionsJson)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(&ssh.SessionConfig{
		Hostname:       *targetOptions.RemoteHostname,
		Port:           *targetOptions.RemotePort,
		Username:       *targetOptions.RemoteUser,
		Password:       targetOptions.RemotePassword,
		PrivateKeyPath: targetOptions.RemotePrivateKey,
	})
}
