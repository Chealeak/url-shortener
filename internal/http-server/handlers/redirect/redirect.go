package redirect

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

//go:generate go run github.com/vektra/mockery/v3@v3.9.1
type GetUrl interface {
	GetUrl(ctx context.Context, alias string) (string, error)
}

func New(log *slog.Logger, urlGetter GetUrl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.redirect.New"

		reqLog := log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")

		url, err := urlGetter.GetUrl(r.Context(), alias)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptyAlias):
				reqLog.Error("Invalid request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.Error("Field alias is a required field"))
			case errors.Is(err, service.ErrAliasNotFound):
				reqLog.Info("Alias not found", slog.String("alias", alias))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.Error("Alias not found"))
			default:
				reqLog.Error("Failed to get URL", slog.String("alias", alias), sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.Error("Failed to get URL"))
			}
			return
		}

		reqLog.Info("got url", slog.String("url", url))

		http.Redirect(w, r, url, http.StatusFound)
	}
}
