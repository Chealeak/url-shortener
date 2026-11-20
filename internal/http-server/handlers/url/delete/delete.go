package delete

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Response struct {
	resp.Response
	CountDeleted int64 `json:"countDeleted"`
}

//go:generate go run github.com/vektra/mockery/v3@v3.9.1
type DeleteUrl interface {
	DeleteUrl(ctx context.Context, alias string) (int64, error)
}

func New(log *slog.Logger, urlRemover DeleteUrl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.delete.New"

		log = log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")

		countDeleted, err := urlRemover.DeleteUrl(r.Context(), alias)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptyAlias):
				log.Error("Invalid request", sl.Err(err))
				render.JSON(w, r, resp.Error("Field alias is a required field"))
			case errors.Is(err, service.ErrURLNotFound):
				log.Info("URL not found", slog.String("alias", alias))
				render.JSON(w, r, resp.Error("URL not found"))
			default:
				log.Error("Failed to delete URL", slog.String("alias", alias), sl.Err(err))
				render.JSON(w, r, resp.Error("Failed to delete URL"))
			}
			return
		}

		log.Info("URL deleted", slog.String("alias", alias), slog.Int64("count_deleted", countDeleted))

		responseOK(w, r, countDeleted)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, countDeleted int64) {
	render.JSON(w, r, Response{
		Response:     resp.OK(),
		CountDeleted: countDeleted,
	})
}
