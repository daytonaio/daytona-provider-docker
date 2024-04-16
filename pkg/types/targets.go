package types

import (
	"encoding/json"

	"github.com/daytonaio/daytona/pkg/provider"
)

type TargetOptions struct {
	RemoteHostname   *string `json:"Remote Hostname,omitempty"`
	RemotePort       *int    `json:"Remote Port,omitempty"`
	RemoteUser       *string `json:"Remote User,omitempty"`
	RemotePassword   *string `json:"Remote Password,omitempty"`
	RemotePrivateKey *string `json:"Remote Private Key Path,omitempty"`
	SockPath         *string `json:"Sock Path,omitempty"`
}

func GetTargetManifest() *provider.ProviderTargetManifest {
	return &provider.ProviderTargetManifest{
		"Remote Hostname": provider.ProviderTargetProperty{
			Type:              provider.ProviderTargetPropertyTypeString,
			DisabledPredicate: "^local$",
		},
		"Remote Port": provider.ProviderTargetProperty{
			Type:              provider.ProviderTargetPropertyTypeInt,
			DefaultValue:      "22",
			DisabledPredicate: "^local$",
		},
		"Remote User": provider.ProviderTargetProperty{
			Type:              provider.ProviderTargetPropertyTypeString,
			DisabledPredicate: "^local$",
		},
		"Remote Password": provider.ProviderTargetProperty{
			Type:              provider.ProviderTargetPropertyTypeString,
			DisabledPredicate: "^local$",
			InputMasked:       true,
		},
		"Remote Private Key Path": provider.ProviderTargetProperty{
			Type:              provider.ProviderTargetPropertyTypeFilePath,
			DefaultValue:      "~/.ssh",
			DisabledPredicate: "^local$",
		},
		"Sock Path": provider.ProviderTargetProperty{
			Type:         provider.ProviderTargetPropertyTypeString,
			DefaultValue: "/var/run/docker.sock",
		},
	}
}

func ParseTargetOptions(optionsJson string) (*TargetOptions, error) {
	var targetOptions TargetOptions
	err := json.Unmarshal([]byte(optionsJson), &targetOptions)
	if err != nil {
		return nil, err
	}

	return &targetOptions, nil
}
