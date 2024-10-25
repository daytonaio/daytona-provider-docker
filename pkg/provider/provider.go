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
	"github.com/daytonaio/daytona/pkg/provider"
	provider_util "github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/ssh"
	"github.com/daytonaio/daytona/pkg/target"
	"github.com/daytonaio/daytona/pkg/target/project"
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
		Name:    "docker-provider",
		Label:   &label,
		Version: internal.Version,
	}, nil
}

func (p DockerProvider) GetTargetConfigManifest() (*provider.TargetConfigManifest, error) {
	return provider_types.GetTargetManifest(), nil
}

func (p DockerProvider) GetPresetTargetConfigs() (*[]provider.TargetConfig, error) {
	info, err := p.GetInfo()
	if err != nil {
		return nil, err
	}

	presetTargets := []provider.TargetConfig{
		{
			Name:         "local",
			ProviderInfo: info,
			Options:      "{\n\t\"Sock Path\": \"/var/run/docker.sock\"\n}",
		},
	}
	return &presetTargets, nil
}

func (p DockerProvider) StartTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
	return new(provider_util.Empty), nil
}

func (p DockerProvider) StopTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
	return new(provider_util.Empty), nil
}

func (p DockerProvider) DestroyTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
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

	err = dockerClient.DestroyTarget(targetReq.Target, targetDir, sshClient)
	if err != nil {
		return new(provider_util.Empty), err
	}

	return new(provider_util.Empty), nil
}

func (p DockerProvider) GetTargetInfo(targetReq *provider.TargetRequest) (*target.TargetInfo, error) {
	dockerClient, err := p.getClient(targetReq.TargetConfigOptions)
	if err != nil {
		return nil, err
	}

	return dockerClient.GetTargetInfo(targetReq.Target)
}

func (p DockerProvider) StartProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetConfigOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	projectDir, err := p.getProjectDir(projectReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		projectLogWriter := loggerFactory.CreateProjectLogger(projectReq.Project.TargetId, projectReq.Project.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, projectLogWriter)
		defer projectLogWriter.Close()
	}

	downloadUrl := *p.DaytonaDownloadUrl
	var sshClient *ssh.Client

	if projectReq.Project.TargetConfig == "local" {
		builderType, err := detect.DetectProjectBuilderType(projectReq.Project.BuildConfig, projectDir, nil)
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
		sshClient, err = p.getSshClient(projectReq.Project.TargetConfig, projectReq.TargetConfigOptions)
		if err != nil {
			return new(provider_util.Empty), err
		}
		defer sshClient.Close()
	}

	err = dockerClient.StartProject(&docker.CreateProjectOptions{
		Project:                  projectReq.Project,
		ProjectDir:               projectDir,
		ContainerRegistry:        projectReq.ContainerRegistry,
		BuilderImage:             projectReq.BuilderImage,
		BuilderContainerRegistry: projectReq.BuilderContainerRegistry,
		LogWriter:                logWriter,
		Gpc:                      projectReq.GitProviderConfig,
		SshClient:                sshClient,
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
	dockerClient, err := p.getClient(projectReq.TargetConfigOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(p.LogsDir, nil)
		projectLogWriter := loggerFactory.CreateProjectLogger(projectReq.Project.TargetId, projectReq.Project.Name, logs.LogSourceProvider)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, projectLogWriter)
		defer projectLogWriter.Close()
	}

	return new(provider_util.Empty), dockerClient.StopProject(projectReq.Project, logWriter)
}

func (p DockerProvider) DestroyProject(projectReq *provider.ProjectRequest) (*provider_util.Empty, error) {
	dockerClient, err := p.getClient(projectReq.TargetConfigOptions)
	if err != nil {
		return new(provider_util.Empty), err
	}

	projectDir, err := p.getProjectDir(projectReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(projectReq.Project.TargetConfig, projectReq.TargetConfigOptions)
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

func (p DockerProvider) GetProjectInfo(projectReq *provider.ProjectRequest) (*project.ProjectInfo, error) {
	dockerClient, err := p.getClient(projectReq.TargetConfigOptions)
	if err != nil {
		return nil, err
	}

	return dockerClient.GetProjectInfo(projectReq.Project)
}

func (p DockerProvider) getClient(targetOptionsJson string) (docker.IDockerClient, error) {
	targetOptions, err := provider_types.ParseTargetConfigOptions(targetOptionsJson)
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

func (p *DockerProvider) getProjectDir(projectReq *provider.ProjectRequest) (string, error) {
	if projectReq.Project.TargetConfig == "local" {
		return filepath.Join(*p.BasePath, projectReq.Project.TargetId, fmt.Sprintf("%s-%s", projectReq.Project.TargetId, projectReq.Project.Name)), nil
	}

	targetOptions, err := provider_types.ParseTargetConfigOptions(projectReq.TargetConfigOptions)
	if err != nil {
		return "", err
	}

	// Using path instead of filepath because we always want to use / as the separator
	return path.Join(*targetOptions.TargetDataDir, projectReq.Project.TargetId, fmt.Sprintf("%s-%s", projectReq.Project.TargetId, projectReq.Project.Name)), nil
}

func (p *DockerProvider) getTargetDir(targetReq *provider.TargetRequest) (string, error) {
	if targetReq.Target.TargetConfig == "local" {
		return filepath.Join(*p.BasePath, targetReq.Target.Id), nil
	}

	targetOptions, err := provider_types.ParseTargetConfigOptions(targetReq.TargetConfigOptions)
	if err != nil {
		return "", err
	}

	// Using path instead of filepath because we always want to use / as the separator
	return path.Join(*targetOptions.TargetDataDir, targetReq.Target.Id), nil
}

func (p *DockerProvider) getSshClient(targetName string, targetOptionsJson string) (*ssh.Client, error) {
	if targetName == "local" {
		return nil, nil
	}

	targetOptions, err := provider_types.ParseTargetConfigOptions(targetOptionsJson)
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
