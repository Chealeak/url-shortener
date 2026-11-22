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

		reqLog := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var request Request
		err := render.DecodeJSON(r.Body, &request)
		if err != nil {
			reqLog.Error("Failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("Failed to decode request body"))
			return
		}

		reqLog.Info("Request body decoded", slog.Any("request", request))

		countUpdated, err := urlUpdater.UpdateUrl(r.Context(), request.Alias, request.NewUrl)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptyAlias):
				reqLog.Error("Invalid request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Field alias is a required field"))
			case errors.Is(err, service.ErrEmptyURL):
				reqLog.Error("Invalid request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Field URL is a required field"))
			case errors.Is(err, service.ErrInvalidURL):
				reqLog.Error("Invalid request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Field URL is not a valid URL"))
			case errors.Is(err, service.ErrAliasNotFound):
				reqLog.Info("Alias not found", slog.String("alias", request.Alias))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.Error("Alias not found"))
			default:
				reqLog.Error("Failed to update URL", slog.String("alias", request.Alias), sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Failed to update URL"))
			}
			return
		}

		reqLog.Info("URL updated", slog.String("alias", request.Alias), slog.Int64("count_updated", countUpdated))

		responseOK(w, r, countUpdated)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, countUpdated int64) {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, Response{
		Response:     resp.OK(),
		CountUpdated: countUpdated,
	})
}
