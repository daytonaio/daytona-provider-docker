package main

import (
	"encoding/gob"
	"os"

	"github.com/daytonaio/daytona/plugins/provisioner"
	provisioner_manager "github.com/daytonaio/daytona/plugins/provisioner/manager"
	"github.com/hashicorp/go-hclog"
	hc_plugin "github.com/hashicorp/go-plugin"

	"provisioner_plugin/plugin"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})
	hc_plugin.Serve(&hc_plugin.ServeConfig{
		HandshakeConfig: provisioner_manager.ProvisionerHandshakeConfig,
		Plugins: map[string]hc_plugin.Plugin{
			"docker-provisioner": &provisioner.ProvisionerPlugin{Impl: &plugin.DockerProvisioner{}},
		},
		Logger: logger,
	})
}

func init() {
	gob.Register(plugin.WorkspaceMetadata{})
	gob.Register(map[string]string{})
}
