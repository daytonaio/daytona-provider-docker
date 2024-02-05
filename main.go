package main

import (
	docker_plugin "provisioner_plugin/plugin"

	"github.com/daytonaio/daytona/plugin"
)

func GetProvisionerPlugin(basePath string) plugin.ProvisionerPlugin {
	return &docker_plugin.DockerProvisionerPlugin{
		BasePath: basePath,
	}
}
