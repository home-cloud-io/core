package web

import (
	"net/http"
	"os"
	"time"

	"github.com/steady-bytes/draft/pkg/chassis"
	"k8s.io/cli-runtime/pkg/printers"

	bugreport "github.com/home-cloud-io/core/pkg/bug-report"
	"github.com/home-cloud-io/core/pkg/kube"
)

type (
	Http interface {
		chassis.RPCRegistrar
	}

	httpHandler struct {
		logger chassis.Logger
		timer  *time.Timer
	}
)

func NewHttp(logger chassis.Logger) Http {
	return &httpHandler{
		logger: logger,
		timer:  time.NewTimer(time.Second),
	}
}

func (h *httpHandler) RegisterRPC(server chassis.Rpcer) {
	server.AddHandler("/bug-report", http.HandlerFunc(h.fileUploadHandler), false)
}

func (h *httpHandler) fileUploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.Info("received request for bug report")

	c, err := kube.NewClient()
	if err != nil {
		h.logger.WithError(err).Error("failed to create kube client")

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	printer := bugreport.Printer{
		Root:         "./bug-report",
		SkippedTypes: []string{},
		Client:       c,
		Printer: &printers.OmitManagedFieldsPrinter{
			Delegate: &printers.YAMLPrinter{},
		},
	}

	select {
	case <-h.timer.C:
		h.logger.Info("creating bug report")
		err = printer.Print(ctx)
		if err != nil {
			h.logger.WithError(err).Error("failed to generate bug report")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		h.logger.Info("creating bug report archive")
		err = printer.Archive()
		if err != nil {
			h.logger.WithError(err).Error("failed to create bug report archive")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// effectively rate limits new reports to every 30s
		h.timer.Reset(30 * time.Second)
	default:
		h.logger.Info("skipping recent report generation")
	}

	f, err := os.ReadFile("./bug-report.tar.gz")
	if err != nil {
		h.logger.WithError(err).Error("failed to read bug report archive")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("serving bug report file")
	w.Header().Set("Content-Disposition", "attachment; filename=bug-report.tar.gz")
	_, err = w.Write(f)
	if err != nil {
		h.logger.WithError(err).Error("failed to serve bug report archive")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.logger.Info("completed request")
}
