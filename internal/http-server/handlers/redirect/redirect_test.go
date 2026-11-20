package redirect_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/redirect/mocks"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service"
)

func TestRedirect_Success(t *testing.T) {
	urlGetterMock := mocks.NewGetUrl(t)
	urlGetterMock.On("GetUrl", mock.Anything, "test_alias").
		Return("https://www.google.com/", nil).
		Once()

	router := chi.NewRouter()
	router.Get("/{alias}", redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock))

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	redirectedToURL, err := api.GetRedirect(server.URL + "/test_alias")
	require.NoError(t, err)
	require.Equal(t, "https://www.google.com/", redirectedToURL)
}

func TestRedirect_NotFound(t *testing.T) {
	urlGetterMock := mocks.NewGetUrl(t)
	urlGetterMock.On("GetUrl", mock.Anything, "missing").
		Return("", service.ErrURLNotFound).
		Once()

	rr := callRedirect(t, urlGetterMock, "/missing")

	require.Equal(t, http.StatusNotFound, rr.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "URL not found", resp.Error)
}

func callRedirect(t *testing.T, getter redirect.GetUrl, target string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Get("/{alias}", redirect.New(slogdiscard.NewDiscardLogger(), getter))

	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}
