package update

import (
	"errors"
	"log/slog"
	"net/http"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Alias  string `json:"alias" validate:"required"`
	NewUrl string `json:"url" validate:"required,url"`
}

type Response struct {
	resp.Response
	CountUpdated int64 `json:"countUpdated"`
}

type UpdateUrl interface {
	UpdateUrl(alias string, newUrl string) (int64, error)
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
		}

		log.Info("Request body decoded", slog.Any("request", request))

		if err := validator.New().Struct(request); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("Invalid request", sl.Err(err))
			render.JSON(w, r, resp.ValidationError(validateErr))
			return
		}

		alias := request.Alias
		newUrl := request.NewUrl

		countUpdated, err := urlUpdater.UpdateUrl(alias, newUrl)
		if errors.Is(err, storage.ErrUrlNotFound) {
			log.Info("URL not found", "alias", alias)
			render.JSON(w, r, resp.Error("url already exists"))
			return
		}

		if err != nil {
			log.Error("Failed to update url", "alias", alias, sl.Err(err))
			render.JSON(w, r, resp.Error("Failed to update url"))
			return
		}

		log.Info("URL updated", slog.String("alias", alias), slog.Int64("count_updated", countUpdated))

		responseOK(w, r, countUpdated)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, countUpdated int64) {
	render.JSON(w, r, Response{
		Response:     resp.OK(),
		CountUpdated: countUpdated,
	})
}
