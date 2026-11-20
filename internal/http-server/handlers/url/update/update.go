package update

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/service"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	Alias  string `json:"alias"`
	NewUrl string `json:"url"`
}

type Response struct {
	resp.Response
	CountUpdated int64 `json:"countUpdated"`
}

type UpdateUrl interface {
	UpdateUrl(ctx context.Context, alias string, newUrl string) (int64, error)
}

func New(log *slog.Logger, urlUpdater UpdateUrl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.update.New"

		log = log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var request Request
		err := render.DecodeJSON(r.Body, &request)
		if err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			render.JSON(w, r, resp.Error("Failed to decode request body"))
			return
		}

		log.Info("Request body decoded", slog.Any("request", request))

		countUpdated, err := urlUpdater.UpdateUrl(r.Context(), request.Alias, request.NewUrl)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptyAlias):
				log.Error("Invalid request", sl.Err(err))
				render.JSON(w, r, resp.Error("Field alias is a required field"))
			case errors.Is(err, service.ErrEmptyURL):
				log.Error("Invalid request", sl.Err(err))
				render.JSON(w, r, resp.Error("Field URL is a required field"))
			case errors.Is(err, service.ErrInvalidURL):
				log.Error("Invalid request", sl.Err(err))
				render.JSON(w, r, resp.Error("Field URL is not a valid URL"))
			case errors.Is(err, service.ErrURLNotFound):
				log.Info("URL not found", slog.String("alias", request.Alias))
				render.JSON(w, r, resp.Error("URL not found"))
			default:
				log.Error("Failed to update URL", slog.String("alias", request.Alias), sl.Err(err))
				render.JSON(w, r, resp.Error("Failed to update URL"))
			}
			return
		}

		log.Info("URL updated", slog.String("alias", request.Alias), slog.Int64("count_updated", countUpdated))

		responseOK(w, r, countUpdated)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, countUpdated int64) {
	render.JSON(w, r, Response{
		Response:     resp.OK(),
		CountUpdated: countUpdated,
	})
}
