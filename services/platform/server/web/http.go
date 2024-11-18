package web

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

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
		file     io.Reader
		volume   string
		path     string
		fileName string
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
	server.AddHandler("/api/upload", http.HandlerFunc(h.fileUploadHandler), true)
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

	id := base64.StdEncoding.EncodeToString([]byte(filepath.Join(form.volume, form.path, form.fileName)))
	fileName := filepath.Join(appRootPath, form.volume, form.path, form.fileName)

	h.logger.WithField("form", form).Info("received form")
	id, err = h.sctl.UploadFileStream(ctx, h.logger, form.file, id, fileName)
	if err != nil {
		h.logger.WithError(err).Error("failed to upload file")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("file upload complete")
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
		case "volume":
			s, err := io.ReadAll(part)
			if err != nil {
				return uploadForm{}, err
			}
			form.volume = string(s)
		case "path":
			s, err := io.ReadAll(part)
			if err != nil {
				return uploadForm{}, err
			}
			form.path = string(s)
		}
	}
	return form, fmt.Errorf("no file provided in form")
}
