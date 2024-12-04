package provider

import (
	"context"
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

	"github.com/daytonaio/daytona/pkg/build/detect"
	"github.com/daytonaio/daytona/pkg/docker"
	"github.com/daytonaio/daytona/pkg/logs"
	"github.com/daytonaio/daytona/pkg/models"
	"github.com/daytonaio/daytona/pkg/provider"
	provider_util "github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/ssh"
	docker_sdk "github.com/docker/docker/client"
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
	label := "Docker"

	return provider.ProviderInfo{
		Name:            "docker-provider",
		Label:           &label,
		AgentlessTarget: true,
		Version:         internal.Version,
	}, nil
}

func (p DockerProvider) GetTargetConfigManifest() (*provider.TargetConfigManifest, error) {
	return provider_types.GetTargetManifest(), nil
}

func (p DockerProvider) GetPresetTargetConfigs() (*[]provider.TargetConfig, error) {
	return &[]provider.TargetConfig{
		{
			Name:    "local",
			Options: "{\n\t\"Sock Path\": \"/var/run/docker.sock\"\n}",
		},
	}, nil
}

func (p DockerProvider) StartTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
	return new(provider_util.Empty), nil
}

func (p DockerProvider) StopTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
	return new(provider_util.Empty), nil
}

func (p DockerProvider) DestroyTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(targetReq.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	targetDir, err := p.getTargetDir(targetReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(targetReq.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}
	if sshClient != nil {
		defer sshClient.Close()
	}

	err = dockerClient.DestroyTarget(targetReq.Target, targetDir, sshClient)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), nil
}

func (p DockerProvider) GetTargetInfo(targetReq *provider.TargetRequest) (*models.TargetInfo, error) {
	dockerClient, err := p.getClient(targetReq.Target.TargetConfig.Options)
	if err != nil {
		return nil, err
	}

	return dockerClient.GetTargetInfo(targetReq.Target)
}

func (p DockerProvider) StartWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	workspaceDir, err := p.getWorkspaceDir(workspaceReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		workspaceLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceReq.Workspace.Id, workspaceReq.Workspace.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, workspaceLogWriter)
		defer workspaceLogWriter.Close()
	}

	downloadUrl := *p.DaytonaDownloadUrl
	var sshClient *ssh.Client

	_, isLocal, err := provider_types.ParseTargetConfigOptions(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	if isLocal {
		builderType, err := detect.DetectWorkspaceBuilderType(workspaceReq.Workspace.BuildConfig, workspaceDir, nil)
		if err != nil {
			return new(provider_util.Empty), err
		}

		if builderType != detect.BuilderTypeDevcontainer {
			parsed, err := url.Parse(downloadUrl)
			if err != nil {
				return new(provider_util.Empty), err
			}

			parsed.Host = fmt.Sprintf("host.docker.internal:%d", *p.ApiPort)
			parsed.Scheme = "http"
			downloadUrl = parsed.String()
		}
	} else {
		sshClient, err = p.getSshClient(workspaceReq.Workspace.Target.TargetConfig.Options)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer sshClient.Close()
	}

	err = dockerClient.StartWorkspace(&docker.CreateWorkspaceOptions{
		Workspace:                workspaceReq.Workspace,
		WorkspaceDir:             workspaceDir,
		ContainerRegistry:        workspaceReq.ContainerRegistry,
		BuilderImage:             workspaceReq.BuilderImage,
		BuilderContainerRegistry: workspaceReq.BuilderContainerRegistry,
		LogWriter:                logWriter,
		Gpc:                      workspaceReq.GitProviderConfig,
		SshClient:                sshClient,
	}, downloadUrl)
	if err != nil {
		return new(provider_util.Empty), err
	}

	go func() {
		err = dockerClient.GetContainerLogs(dockerClient.GetWorkspaceContainerName(workspaceReq.Workspace), logWriter)
		if err != nil {
			logWriter.Write([]byte(err.Error()))
		}
	}()

	return new(provider_util.Empty), nil
}

func (p DockerProvider) StopWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		workspaceLogWriter := loggerFactory.CreateWorkspaceLogger(workspaceReq.Workspace.Id, workspaceReq.Workspace.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, workspaceLogWriter)
		defer workspaceLogWriter.Close()
	}

	return new(provider_util.Empty), dockerClient.StopWorkspace(workspaceReq.Workspace, logWriter)
}

func (p DockerProvider) DestroyWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	workspaceDir, err := p.getWorkspaceDir(workspaceReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(workspaceReq.Workspace.Target.TargetConfig.Options)
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

func (p DockerProvider) GetWorkspaceInfo(workspaceReq *provider.WorkspaceRequest) (*models.WorkspaceInfo, error) {
	dockerClient, err := p.getClient(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return nil, err
	}

	return dockerClient.GetWorkspaceInfo(workspaceReq.Workspace)
}

func (p DockerProvider) getClient(targetOptionsJson string) (docker.IDockerClient, error) {
	targetOptions, _, err := provider_types.ParseTargetConfigOptions(targetOptionsJson)
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

func (p DockerProvider) CheckRequirements() (*[]provider.RequirementStatus, error) {
	var results []provider.RequirementStatus
	ctx := context.Background()

	cli, err := docker_sdk.NewClientWithOpts(docker_sdk.FromEnv, docker_sdk.WithAPIVersionNegotiation())
	if err != nil {
		results = append(results, provider.RequirementStatus{
			Name:   "Docker installed",
			Met:    false,
			Reason: "Docker is not installed",
		})
		return &results, nil
	} else {
		results = append(results, provider.RequirementStatus{
			Name:   "Docker installed",
			Met:    true,
			Reason: "Docker is installed",
		})
	}

	// Check if Docker is running by fetching Docker info
	_, err = cli.Info(ctx)
	if err != nil {
		results = append(results, provider.RequirementStatus{
			Name:   "Docker running",
			Met:    false,
			Reason: "Docker is not running. Error: " + err.Error(),
		})
	} else {
		results = append(results, provider.RequirementStatus{
			Name:   "Docker running",
			Met:    true,
			Reason: "Docker is running",
		})
	}
	return &results, nil
}

func (p *DockerProvider) getWorkspaceDir(workspaceReq *provider.WorkspaceRequest) (string, error) {
	targetOptions, isLocal, err := provider_types.ParseTargetConfigOptions(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return "", err
	}

	if isLocal {
		return filepath.Join(*p.BasePath, workspaceReq.Workspace.Id, workspaceReq.Workspace.WorkspaceFolderName()), nil
	}

	// Using path instead of filepath because we always want to use / as the separator
	return path.Join(*targetOptions.TargetDataDir, workspaceReq.Workspace.Id, workspaceReq.Workspace.WorkspaceFolderName()), nil
}

func (p *DockerProvider) getTargetDir(targetReq *provider.TargetRequest) (string, error) {
	targetOptions, isLocal, err := provider_types.ParseTargetConfigOptions(targetReq.Target.TargetConfig.Options)
	if err != nil {
		return "", err
	}

	if isLocal {
		return filepath.Join(*p.BasePath, targetReq.Target.Id), nil
	}

	// Using path instead of filepath because we always want to use / as the separator
	return path.Join(*targetOptions.TargetDataDir, targetReq.Target.Id), nil
}

func (p *DockerProvider) getSshClient(targetOptionsJson string) (*ssh.Client, error) {
	targetOptions, isLocal, err := provider_types.ParseTargetConfigOptions(targetOptionsJson)
	if err != nil {
		return nil, err
	}

	if isLocal {
		return nil, nil
	}

	return ssh.NewClient(&ssh.SessionConfig{
		Hostname:       *targetOptions.RemoteHostname,
		Port:           *targetOptions.RemotePort,
		Username:       *targetOptions.RemoteUser,
		Password:       targetOptions.RemotePassword,
		PrivateKeyPath: targetOptions.RemotePrivateKey,
	})
}
