package provider_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/daytonaio/daytona-provider-docker/pkg/client"

	"github.com/daytonaio/daytona/pkg/docker"
	"github.com/daytonaio/daytona/pkg/gitprovider"
	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/daytonaio/daytona/pkg/target"
	"github.com/daytonaio/daytona/pkg/target/project"

	docker_provider "github.com/daytonaio/daytona-provider-docker/pkg/provider"
	provider_types "github.com/daytonaio/daytona-provider-docker/pkg/types"

	docker_types "github.com/docker/docker/api/types"
	docker_client "github.com/docker/docker/client"
)

var dockerProvider = &docker_provider.DockerProvider{}
var targetOptions = &provider_types.TargetConfigOptions{}
var sockDir = "/tmp/target-socks"
var optionsString string

var project1 = &project.Project{
	Name: "test",
	Repository: &gitprovider.GitRepository{
		Id:   "123",
		Url:  "https://github.com/daytonaio/daytona",
		Name: "daytona",
	},
	Image:    "daytonaio/target-project:latest",
	TargetId: "123",
}

var target1 = &target.Target{
	Id:           "123",
	Name:         "test",
	TargetConfig: "local",
	Projects: []*project.Project{
		project1,
	},
}

func GetContainerName(project *project.Project) string {
	dockerClient := docker.NewDockerClient(docker.DockerClientConfig{})

	return dockerClient.GetProjectContainerName(project)
}

func TestCreateTarget(t *testing.T) {
	targetReq := &provider.TargetRequest{
		TargetConfigOptions: optionsString,
		Target:              target1,
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
		TargetConfigOptions: optionsString,
		Target:              target1,
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
		TargetConfigOptions: optionsString,
		Target:              target1,
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

func TestCreateProject(t *testing.T) {
	TestCreateTarget(t)

	projectReq := &provider.ProjectRequest{
		TargetConfigOptions: optionsString,
		Project:             project1,
	}

	_, err := dockerProvider.CreateProject(projectReq)
	if err != nil {
		t.Errorf("Error creating project: %s", err)
	}

	_, err = getDockerClient().ContainerInspect(context.Background(), GetContainerName(project1))
	if err != nil {
		t.Errorf("Expected container to exist")
	}
}

func TestDestroyProject(t *testing.T) {
	projectReq := &provider.ProjectRequest{
		TargetConfigOptions: optionsString,
		Project:             project1,
	}

	_, err := dockerProvider.DestroyProject(projectReq)
	if err != nil {
		t.Errorf("Error deleting project: %s", err)
	}

	_, err = getDockerClient().ContainerInspect(context.Background(), GetContainerName(project1))
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
}
