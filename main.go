package main

import (
	"os"

	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/daytonaio/daytona/pkg/runner/providermanager"
	"github.com/hashicorp/go-hclog"
	hc_plugin "github.com/hashicorp/go-plugin"

	p "github.com/daytonaio/daytona-provider-docker/pkg/provider"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})
	hc_plugin.Serve(&hc_plugin.ServeConfig{
		HandshakeConfig: providermanager.ProviderHandshakeConfig,
		Plugins: map[string]hc_plugin.Plugin{
			"docker-provider": &provider.ProviderPlugin{Impl: &p.DockerProvider{}},
		},
		Logger: logger,
	})
}
