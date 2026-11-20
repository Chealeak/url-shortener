package save

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
	URL   string `json:"url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

//go:generate go run github.com/vektra/mockery/v3@v3.9.1
type SaveUrl interface {
	SaveUrl(ctx context.Context, url, alias string) (string, error)
}

func New(log *slog.Logger, urlSaver SaveUrl) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.save.New"

		log = log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			render.JSON(w, r, resp.Error("Failed to decode request body"))
			return
		}

		log.Info("Request body decoded", slog.Any("request", req))

		alias, err := urlSaver.SaveUrl(r.Context(), req.URL, req.Alias)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptyURL):
				log.Error("Invalid request", sl.Err(err))
				render.JSON(w, r, resp.Error("Field URL is a required field"))
			case errors.Is(err, service.ErrInvalidURL):
				log.Error("Invalid request", sl.Err(err))
				render.JSON(w, r, resp.Error("Field URL is not a valid URL"))
			case errors.Is(err, service.ErrURLExists):
				log.Info("URL already exists", slog.String("url", req.URL))
				render.JSON(w, r, resp.Error("URL already exists"))
			default:
				log.Error("Failed to add URL", sl.Err(err))
				render.JSON(w, r, resp.Error("Failed to add URL"))
			}
			return
		}

		log.Info("URL added", slog.String("alias", alias))

		responseOK(w, r, alias)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
