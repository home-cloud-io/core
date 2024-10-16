package web

import (
	"bufio"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	v1 "github.com/home-cloud-io/core/api/platform/server/v1"
	"github.com/home-cloud-io/core/services/platform/server/apps"
	"github.com/home-cloud-io/core/services/platform/server/system"

	"github.com/steady-bytes/draft/pkg/chassis"
)

type (
	Http interface {
		chassis.RPCRegistrar
	}

	httpHandler struct {
		logger chassis.Logger
		actl   apps.Controller
		sctl   system.Controller
	}

	uploadForm struct {
		file             io.Reader
		app              string
		path             string
		fileName         string
		fileNameOverride string
		id               string
	}
)

const (
	appRootPath = "/mnt/k8s-pvs"
)

func NewHttp(logger chassis.Logger, actl apps.Controller, sctl system.Controller) Http {
	return &httpHandler{
		logger,
		actl,
		sctl,
	}
}

// Implement the `RPCRegistrar` interface of draft so the `grpc` handlers are enabled
func (h *httpHandler) RegisterRPC(server chassis.Rpcer) {
	server.AddHandler("/upload-file", http.HandlerFunc(h.fileUploadHandler), true)
}

func (h *httpHandler) fileUploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Info("receiving file for upload")

	// read all parts of form
	reader, err := r.MultipartReader()
	if err != nil {
		h.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	form, err := readForm(reader)
	if err != nil {
		h.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileName := form.fileName
	if form.fileNameOverride != "" {
		fileName = form.fileNameOverride
	}
	fileName = filepath.Join(appRootPath, form.app, form.path, fileName)

	h.logger.WithField("form", form).Info("received form")
	id, err := h.sctl.UploadFileStream(ctx, h.logger, form.file, form.id, fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("file upload complete")
	err = events.Send(&v1.ServerEvent{
		Event: &v1.ServerEvent_FileUploaded{
			FileUploaded: &v1.FileUploadedEvent{
				Id: id,
			},
		},
	})
	if err != nil {
		h.logger.WithError(err).Error("failed to send file uploaded event to client")
		return
	}
	h.logger.Info("sent uploaded event to client")
}

func readForm(reader *multipart.Reader) (uploadForm, error) {
	form := uploadForm{}
	for {
		part, err := reader.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			return uploadForm{}, err
		}

		switch part.FormName() {
		case "file":
			form.file = bufio.NewReader(part)
			form.fileName = part.FileName()
			return form, nil
		case "app":
			s, err := io.ReadAll(part)
			if err != nil {
				return uploadForm{}, err
			}
			form.app = string(s)
		case "path":
			s, err := io.ReadAll(part)
			if err != nil {
				return uploadForm{}, err
			}
			form.path = string(s)
		case "file-name-override":
			s, err := io.ReadAll(part)
			if err != nil {
				return uploadForm{}, err
			}
			form.fileNameOverride = string(s)
		case "id":
			s, err := io.ReadAll(part)
			if err != nil {
				return uploadForm{}, err
			}
			form.id = string(s)
		}
	}
	return form, fmt.Errorf("no file provided in form")
}
