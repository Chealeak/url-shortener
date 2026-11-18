package delete

import (
	"log/slog"
	"net/http"
	resp "url-shortener/internal/lib/api/response"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

// TODO: move to config
const aliasLength = 6

//go:generate go run github.com/vektra/mockery/v3@v3.9.1
type URLRemover interface {
	DeleteURL(alias string) (int64, error)
}

func New(log *slog.Logger, urlRemover URLRemover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.delete.New"

		log = log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Error("empty alias")
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		_, err := urlRemover.DeleteURL(alias)
		if err != nil {
			log.Error("failed to remove url", "alias", alias)
			render.JSON(w, r, resp.Error("failed to remove url"))
			return
		}

		log.Info("url removed", slog.String("alias", alias))

		responseOK(w, r, alias)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
