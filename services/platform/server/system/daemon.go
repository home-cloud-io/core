package system

import (
	"context"
	"io"

	dv1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/server/async"
	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Daemon interface {
		// ShutdownHost will shutdown the host machine running Home Cloud.
		ShutdownHost() error
		// RestartHost will restart the host machine running Home Cloud
		RestartHost() error
		// ChangeDaemonVersion will update the NixOS config with a new Daemon version
		// and switch to it.
		ChangeDaemonVersion(cmd *dv1.ChangeDaemonVersionCommand) error
		// AddMdnsHost adds a host to the avahi mDNS server managed by the daemon
		AddMdnsHost(hostname string) error
		// RemoveMdnsHost removes a host to the avahi mDNS server managed by the daemon
		RemoveMdnsHost(hostname string) error
		// UploadFileStream will stream a file in chunks as an upload to the daemon
		UploadFileStream(ctx context.Context, logger chassis.Logger, buf io.Reader, fileId, fileName string) (string, error)
	}
)

// DAEMON

func (c *controller) ShutdownHost() error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_Shutdown{},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) RestartHost() error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_Restart{},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) ChangeDaemonVersion(cmd *dv1.ChangeDaemonVersionCommand) error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_ChangeDaemonVersionCommand{
			ChangeDaemonVersionCommand: cmd,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) AddMdnsHost(hostname string) error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_AddMdnsHostCommand{
			AddMdnsHostCommand: &dv1.AddMdnsHostCommand{
				Hostname: hostname,
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) RemoveMdnsHost(hostname string) error {
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_RemoveMdnsHostCommand{
			RemoveMdnsHostCommand: &dv1.RemoveMdnsHostCommand{
				Hostname: hostname,
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *controller) UploadFileStream(ctx context.Context, logger chassis.Logger, buf io.Reader, fileId, fileName string) (string, error) {
	logger.Info("uploading file")
	listener := async.RegisterListener(ctx, c.broadcaster, &async.ListenerOptions[*dv1.UploadFileReady]{
		Callback: func(event *dv1.UploadFileReady) (bool, error) {
			if event.Id == fileId {
				return true, nil
			}
			return false, nil
		},
	})

	// prepare upload to daemon
	err := com.Send(&dv1.ServerMessage{
		Message: &dv1.ServerMessage_UploadFileRequest{
			UploadFileRequest: &dv1.UploadFileRequest{
				Data: &dv1.UploadFileRequest_Info{
					Info: &dv1.FileInfo{
						FileId:   fileId,
						FilePath: fileName,
					},
				},
			},
		},
	})
	if err != nil {
		logger.WithError(err).Error("failed to ready daemon for file upload")
		return fileId, err
	}
	logger.Info("waiting for ready signal")
	err = listener.Listen(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to ready daemon for file upload")
		return fileId, err
	}
	logger.Info("daemon ready for file upload")

	// chunk file and upload
	err = c.streamFile(ctx, logger, buf, fileId, fileName)
	if err != nil {
		logger.WithError(err).Error("failed to upload chunked file")
		return fileId, err
	}

	return fileId, nil
}
