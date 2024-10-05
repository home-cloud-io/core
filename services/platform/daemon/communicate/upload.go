package communicate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	v1 "github.com/home-cloud-io/core/api/platform/daemon/v1"
	"github.com/steady-bytes/draft/pkg/chassis"
)

const (
	chunkPath     = "/etc/daemon/tmp"
)

func (c *client) uploadFile(_ context.Context, def *v1.UploadFileRequest) {
	switch def.Data.(type) {
	case *v1.UploadFileRequest_Info:
		info := def.GetInfo()
		log := c.logger.WithFields(chassis.Fields{
			"file_id":   info.FileId,
			"file_path": info.FilePath,
		})
		// check if existing upload exists and ignore if so (only a single info message should be sent for a given file)
		if _, ok := c.fileMetas[info.FileId]; ok {
			log.Warn("info message recieved for already instantiated upload buffer")
			return
		}

		// save metadata
		c.fileMetas[info.FileId] = fileMeta{
			id:       info.FileId,
			filePath: info.FilePath,
		}

		// make temporary chunk upload directory
		err := os.MkdirAll(filepath.Join(chunkPath, info.FileId), 0777)
		if err != nil {
			log.WithError(err).Error("failed to create temp directory for file upload")
			return
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
			return
		}
		log.Info("completed file upload setup")
	case *v1.UploadFileRequest_Chunk:
		chunk := def.GetChunk()
		log := c.logger.WithField("file_id", chunk.FileId).WithField("chunk_index", chunk.Index)
		log.Debug("processing chunk")

		// check if existing upload exists and ignore if not
		meta, ok := c.fileMetas[chunk.FileId]
		if !ok {
			log.Warn("chunk message received for uninitiated file upload")
			return
		}
		log = log.WithFields(chassis.Fields{
			"file_path": meta.filePath,
		})

		// write chunk as temp file
		fileName := filepath.Join(chunkPath, meta.id, fmt.Sprintf("chunk.%d", chunk.Index))
		err := os.WriteFile(fileName, chunk.Data, 0666)
		if err != nil {
			log.WithError(err).Error("failed to write uploaded file chunk")
			return
		}
		log.Debug("wrote temp file")

		// store chunk meta
		c.chunkMetas.Store(fmt.Sprintf("%s.%d", chunk.FileId, chunk.Index), chunkMeta{
			index:    chunk.Index,
			fileName: fileName,
		})
		log.Debug("saved chunk metadata")

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
			return
		}

		log.Debug("completed writing chunk")

	case *v1.UploadFileRequest_Done:
		done := def.GetDone()
		log := c.logger.WithField("file_id", done.FileId)

		// check if existing upload exists and ignore if not
		meta, ok := c.fileMetas[done.FileId]
		if !ok {
			log.Warn("done message received for uninitiated file upload")
			return
		}
		log = log.WithFields(chassis.Fields{
			"file_path": meta.filePath,
		})

		err := c.reconstructFile(log, meta)
		if err != nil {
			log.WithError(err).Error("failed to reconstruct file from chunks")
			return
		}

		log.Info("completed saving file")

	default:
		c.logger.Error("unknown UploadFileRequest type")
	}
}

// reconstructFile reconstructs a file from its chunks using the provided metadata.
// It reads each chunk file, concatenates their content, and writes it to the output file.
func (c *client) reconstructFile(logger chassis.Logger, metadata fileMeta) error {
	// create parent directories if not exists
	err := os.MkdirAll(filepath.Dir(metadata.filePath), 0777)
	if err != nil {
		return err
	}

	// create final file
	outputFile, err := os.Create(metadata.filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// extract (and sort) chunks from metadata
	var chunks []chunkMeta
	i := 0
	for {
		chunk, ok := c.chunkMetas.LoadAndDelete(fmt.Sprintf("%s.%d", metadata.id, i))
		if !ok {
			break
		}
		chunks = append(chunks, chunk.(chunkMeta))
		i++
	}

	// iterate through the sorted chunks and concatenate their content to build the final file
	for _, chunk := range chunks {
		// open the chunk file
		chunkFile, err := os.Open(chunk.fileName)
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
		err = os.Remove(chunk.fileName)
		if err != nil {
			logger.WithError(err).Error("failed to remove chunk file")
		}
	}

	// remove the chunk file directory
	err = os.Remove(filepath.Join(chunkPath, metadata.id))
	if err != nil {
		logger.WithError(err).Error("failed to remove chunk directory")
	}

	return nil
}
