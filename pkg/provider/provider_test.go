package provider_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/daytonaio/daytona-provider-docker/pkg/client"

	"github.com/daytonaio/daytona/pkg/docker"
	"github.com/daytonaio/daytona/pkg/gitprovider"
	"github.com/daytonaio/daytona/pkg/models"
	"github.com/daytonaio/daytona/pkg/provider"

	docker_provider "github.com/daytonaio/daytona-provider-docker/pkg/provider"
	provider_types "github.com/daytonaio/daytona-provider-docker/pkg/types"

	docker_types "github.com/docker/docker/api/types"
	docker_client "github.com/docker/docker/client"
)

var dockerProvider = &docker_provider.DockerProvider{}
var targetOptions = &provider_types.TargetConfigOptions{}
var sockDir = "/tmp/target-socks"
var optionsString string

var workspace1 = &models.Workspace{
	Name: "test",
	Repository: &gitprovider.GitRepository{
		Id:   "123",
		Url:  "https://github.com/daytonaio/daytona",
		Name: "daytona",
	},
	Image:    "daytonaio/workspace-project:latest",
	TargetId: "123",
}

var targetConfig1 = &models.TargetConfig{
	Id:   "test",
	Name: "test",
	ProviderInfo: models.ProviderInfo{
		Name:    "docker-provider",
		Version: "test",
	},
	Options: "test-options",
	Deleted: false,
}

var target1 = &models.Target{
	Id:             "123",
	Name:           "test",
	TargetConfigId: targetConfig1.Id,
	TargetConfig:   *targetConfig1,
}

func GetContainerName(workspace *models.Workspace) string {
	dockerClient := docker.NewDockerClient(docker.DockerClientConfig{})

	return dockerClient.GetWorkspaceContainerName(workspace)
}

func TestCreateTarget(t *testing.T) {
	targetReq := &provider.TargetRequest{
		Target: target1,
	}

	_, err := dockerProvider.CreateTarget(targetReq)
	if err != nil {
		t.Errorf("Error creating target: %s", err)
	}

	_, err = getDockerClient().NetworkInspect(context.Background(), target1.Id, docker_types.NetworkInspectOptions{})
	if err != nil {
		t.Errorf("Expected network to exist")
	}
}

func TestGetTargetInfo(t *testing.T) {
	targetReq := &provider.TargetRequest{
		Target: target1,
	}

	targetInfo, err := dockerProvider.GetTargetInfo(targetReq)
	if err != nil || targetInfo == nil {
		t.Errorf("Error getting target info: %s", err)
	}

	var targetMetadata provider_types.TargetMetadata
	err = json.Unmarshal([]byte(targetInfo.ProviderMetadata), &targetMetadata)
	if err != nil {
		t.Errorf("Error unmarshalling target metadata: %s", err)
	}

	if targetMetadata.NetworkId != targetReq.Target.Id {
		t.Errorf("Expected network id %s, got %s", targetReq.Target.Id, targetMetadata.NetworkId)
	}
}

func TestDestroyTarget(t *testing.T) {
	targetReq := &provider.TargetRequest{
		Target: target1,
	}

	_, err := dockerProvider.DestroyTarget(targetReq)
	if err != nil {
		t.Errorf("Error deleting target: %s", err)
	}

	dockerClient, err := client.GetClient(*targetOptions, sockDir)
	if err != nil {
		t.Errorf("Error getting docker client: %s", err)
	}

	_, err = dockerClient.NetworkInspect(context.Background(), target1.Id, docker_types.NetworkInspectOptions{})
	if err == nil {
		t.Errorf("Expected network to not exist")
	}
}

func TestCreateWorkspace(t *testing.T) {
	TestCreateTarget(t)

	workspaceReq := &provider.WorkspaceRequest{
		Workspace: workspace1,
	}

	_, err := dockerProvider.CreateWorkspace(workspaceReq)
	if err != nil {
		t.Errorf("Error creating workspace: %s", err)
	}

	_, err = getDockerClient().ContainerInspect(context.Background(), GetContainerName(workspace1))
	if err != nil {
		t.Errorf("Expected container to exist")
	}
}

func TestDestroyWorkspace(t *testing.T) {
	workspaceReq := &provider.WorkspaceRequest{
		Workspace: workspace1,
	}

	_, err := dockerProvider.DestroyWorkspace(workspaceReq)
	if err != nil {
		t.Errorf("Error deleting workspace: %s", err)
	}

	_, err = getDockerClient().ContainerInspect(context.Background(), GetContainerName(workspace1))
	if err == nil {
		t.Errorf("Expected container to not exist")
	}

	TestDestroyTarget(t)
}

func getDockerClient() *docker_client.Client {
	dockerClient, err := client.GetClient(*targetOptions, sockDir)
	if err != nil {
		panic(err)
	}

	return dockerClient
}

func init() {
	_, err := dockerProvider.Initialize(provider.InitializeProviderRequest{
		BasePath:           "/tmp/targets",
		DaytonaDownloadUrl: "https://download.daytona.io/daytona/get-server.sh",
		DaytonaVersion:     "latest",
		ServerUrl:          "",
		ApiUrl:             "",
		ServerPort:         0,
		ApiPort:            0,
	})
	if err != nil {
		panic(err)
	}

	opts, err := json.Marshal(targetOptions)
	if err != nil {
		panic(err)
	}

	optionsString = string(opts)
	target1.TargetConfig.Options = optionsString
}
