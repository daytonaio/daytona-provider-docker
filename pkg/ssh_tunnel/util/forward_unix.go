package util

import (
	"context"
	"errors"

	"provider/pkg/ssh_tunnel"
	"provider/pkg/types"

	log "github.com/sirupsen/logrus"
)

func ForwardRemoteUnixSock(ctx context.Context, targetOptions types.TargetOptions, localSock string, remoteSock string) (chan bool, chan error) {
	if targetOptions.RemoteHostname == nil {
		errChan := make(chan error)
		errChan <- errors.New("Remote Hostname is required")
		return make(chan bool), errChan
	}

	sshTun := ssh_tunnel.NewUnix(localSock, *targetOptions.RemoteHostname, remoteSock)

	if targetOptions.RemotePort != nil {
		sshTun.SetPort(*targetOptions.RemotePort)
	}
	if targetOptions.RemoteUser != nil {
		sshTun.SetUser(*targetOptions.RemoteUser)
	}

	if targetOptions.RemotePassword != nil {
		sshTun.SetPassword(*targetOptions.RemotePassword)
	} else if targetOptions.RemotePrivateKey != nil {
		privateKeyPath, password, err := GetSshPrivateKeyPath(*targetOptions.RemotePrivateKey)
		if err != nil {
			log.Fatal(err)
		}
		if password != nil {
			sshTun.SetEncryptedKeyFile(privateKeyPath, *password)
		} else {
			sshTun.SetKeyFile(privateKeyPath)
		}
	}

	errChan := make(chan error)

	sshTun.SetTunneledConnState(func(tun *ssh_tunnel.SshTunnel, state *ssh_tunnel.TunneledConnectionState) {
		log.Debugf("%+v", state)
	})

	startedChann := make(chan bool, 1)

	sshTun.SetConnState(func(tun *ssh_tunnel.SshTunnel, state ssh_tunnel.ConnectionState) {
		switch state {
		case ssh_tunnel.StateStarting:
			log.Debugf("SSH Tunnel is Starting")
		case ssh_tunnel.StateStarted:
			log.Debugf("SSH Tunnel is Started")
			startedChann <- true
		case ssh_tunnel.StateStopped:
			log.Debugf("SSH Tunnel is Stopped")
		}
	})

	go func() {
		errChan <- sshTun.Start(ctx)
	}()

	return startedChann, errChan
}
