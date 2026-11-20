package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/storage"

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

		url, err := urlGetter.GetUrl(r.Context(), alias)
		if errors.Is(err, storage.ErrUrlNotFound) {
			log.Info("url not found", "alias", alias)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}
		if err != nil {
			log.Error("failed to get url", "alias", alias)
			render.JSON(w, r, resp.Error("failed to get url"))
			return
		}

		log.Info("got url", slog.String("url", url))

		http.Redirect(w, r, url, http.StatusFound)
	}
}
