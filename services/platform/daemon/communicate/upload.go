package communicate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/home-cloud-io/core/services/platform/daemon/host"
	"github.com/steady-bytes/draft/pkg/chassis"
)

func (c *client) uploadFile(_ context.Context, def *v1.UploadFileRequest) {
	switch def.Data.(type) {
	case *v1.UploadFileRequest_Info:
		info := def.GetInfo()
		info.FilePath = host.FilePath(info.FilePath)
		log := c.logger.WithFields(chassis.Fields{
			"file_id":   info.FileId,
			"file_path": info.FilePath,
		})

		chunkPath := filepath.Join(host.ChunkPath(), info.FileId)

		// create tmp chunk path if not exists
		_, err := os.Stat(chunkPath)
		if os.IsNotExist(err) {
			// make temporary chunk upload directory
			err := os.MkdirAll(filepath.Join(host.ChunkPath(), info.FileId), 0777)
			if err != nil {
				log.WithError(err).Error("failed to create temp directory for file upload")
				cleanupFailedFileUpload(log, info.FileId)
				return
			}
		}

		// repond to server with ready
		err = c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_UploadFileReady{
				UploadFileReady: &v1.UploadFileReady{
					Id: info.FileId,
				},
			},
		})
		if err != nil {
			log.Error("failed to alert server with ready state for file upload")
			cleanupFailedFileUpload(log, info.FileId)
			return
		}
		log.Info("completed file upload setup")
	case *v1.UploadFileRequest_Chunk:
		chunk := def.GetChunk()
		log := c.logger.WithFields(chassis.Fields{
			"file_id":     chunk.FileId,
			"chunk_index": chunk.Index,
		})
		log.Debug("processing chunk")

		chunkPath := filepath.Join(host.ChunkPath(), chunk.FileId)

		// check if existing upload exists and ignore if not
		_, err := os.Stat(chunkPath)
		if os.IsNotExist(err) {
			log.Warn("chunk message received for uninitiated file upload")
			return
		}

		// write chunk as temp file
		fileName := filepath.Join(host.ChunkPath(), chunk.FileId, fmt.Sprintf("chunk.%d", chunk.Index))
		err = os.WriteFile(fileName, chunk.Data, 0666)
		if err != nil {
			log.WithError(err).Error("failed to write uploaded file chunk")
			cleanupFailedFileUpload(log, chunk.FileId)
			return
		}
		log.Debug("wrote temp file")

		// repond to server with chunk completion
		err = c.Send(&v1.DaemonMessage{
			Message: &v1.DaemonMessage_UploadFileChunkCompleted{
				UploadFileChunkCompleted: &v1.UploadFileChunkCompleted{
					FileId: chunk.FileId,
					Index:  chunk.Index,
				},
			},
		})
		if err != nil {
			log.WithError(err).Error("failed to alert server with completed state for chunk upload")
			cleanupFailedFileUpload(log, chunk.FileId)
			return
		}

		log.Debug("completed writing chunk")

	case *v1.UploadFileRequest_Done:
		done := def.GetDone()
		log := c.logger.WithField("file_id", done.FileId)

		// check if existing upload exists and ignore if not
		chunkPath := filepath.Join(host.ChunkPath(), done.FileId)
		_, err := os.Stat(chunkPath)
		if os.IsNotExist(err) {
			log.Warn("chunk message received for uninitiated file upload")
			return
		}

		err = reconstructFile(log, done.FileId, done.FilePath)
		if err != nil {
			log.WithError(err).Error("failed to reconstruct file from chunks")
			cleanupFailedFileUpload(log, done.FileId)
			return
		}

		log.Info("completed saving file")

	default:
		c.logger.Error("unknown UploadFileRequest type")
	}
}

func cleanupFailedFileUpload(logger chassis.Logger, fileId string) {
	// remove chunk directory
	err := os.RemoveAll(filepath.Join(host.ChunkPath(), fileId))
	if err != nil {
		logger.WithError(err).Error("failed to remove chunk directory")
	}
}

// reconstructFile reconstructs a file from its chunks using the provided metadata.
// It reads each chunk file, concatenates their content, and writes it to the output file.
func reconstructFile(logger chassis.Logger, fileId string, filePath string) error {
	filePath = host.FilePath(filePath)

	// create parent directories if not exists
	err := os.MkdirAll(filepath.Dir(filePath), 0777)
	if err != nil {
		return err
	}

	// create final file
	outputFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	chunks, err := os.ReadDir(filepath.Join(host.ChunkPath(), fileId))
	if err != nil {
		logger.WithError(err).Error("failed to read chunk directory")
		return err
	}

	// iterate through the sorted chunks and concatenate their content to build the final file
	for _, chunk := range chunks {
		chunkPath := filepath.Join(host.ChunkPath(), fileId, chunk.Name())

		// open the chunk file
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		defer chunkFile.Close()

		// copy the chunk into the final file
		_, err = io.Copy(outputFile, chunkFile)
		if err != nil {
			return err
		}
		err = chunkFile.Close()
		if err != nil {
			logger.WithError(err).Error("failed to close chunk file")
		}

		// remove the chunk file
		err = os.Remove(chunkPath)
		if err != nil {
			logger.WithError(err).Error("failed to remove chunk file")
		}
	}

	// remove the chunk file directory
	err = os.Remove(filepath.Join(host.ChunkPath(), fileId))
	if err != nil {
		logger.WithError(err).Error("failed to remove chunk directory")
	}

	return nil
}
