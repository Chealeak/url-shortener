package save_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service"
)

func TestSave_Success(t *testing.T) {
	urlSaverMock := mocks.NewSaveUrl(t)
	urlSaverMock.On("SaveUrl", mock.Anything, "https://google.com", "test_alias").
		Return("test_alias", nil).
		Once()

	rr := callSave(t, urlSaverMock, `{"url":"https://google.com","alias":"test_alias"}`)

	require.Equal(t, http.StatusCreated, rr.Code)
	resp := decodeSave(t, rr)
	require.Empty(t, resp.Error)
	require.Equal(t, "test_alias", resp.Alias)
}

func TestSave_BadRequest(t *testing.T) {
	urlSaverMock := mocks.NewSaveUrl(t)

	rr := callSave(t, urlSaverMock, `{"url":`)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	resp := decodeSave(t, rr)
	require.Equal(t, "Failed to decode request body", resp.Error)
	require.Empty(t, resp.Alias)
}

func TestSave_InvalidAlias(t *testing.T) {
	urlSaverMock := mocks.NewSaveUrl(t)
	urlSaverMock.On("SaveUrl", mock.Anything, "https://google.com", "test/example").
		Return("", service.ErrInvalidAlias).
		Once()

	rr := callSave(t, urlSaverMock, `{"url":"https://google.com","alias":"test/example"}`)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	resp := decodeSave(t, rr)
	require.Equal(t, "Field alias must contain only letters, digits, '-' and '_' (up to 32 characters)", resp.Error)
	require.Empty(t, resp.Alias)
}

func TestSave_Conflict(t *testing.T) {
	urlSaverMock := mocks.NewSaveUrl(t)
	urlSaverMock.On("SaveUrl", mock.Anything, "https://google.com", "test_alias").
		Return("", service.ErrURLExists).
		Once()

	rr := callSave(t, urlSaverMock, `{"url":"https://google.com","alias":"test_alias"}`)

	require.Equal(t, http.StatusConflict, rr.Code)
	resp := decodeSave(t, rr)
	require.Equal(t, "URL already exists", resp.Error)
}

func TestSave_InternalError(t *testing.T) {
	urlSaverMock := mocks.NewSaveUrl(t)
	urlSaverMock.On("SaveUrl", mock.Anything, "https://google.com", "test_alias").
		Return("", errors.New("db is down")).
		Once()

	rr := callSave(t, urlSaverMock, `{"url":"https://google.com","alias":"test_alias"}`)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	resp := decodeSave(t, rr)
	require.Equal(t, "Failed to add URL", resp.Error)
}

func callSave(t *testing.T, urlSaver save.SaveUrl, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := save.New(slogdiscard.NewDiscardLogger(), urlSaver)
	req := httptest.NewRequest(http.MethodPost, "/url", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func decodeSave(t *testing.T, rr *httptest.ResponseRecorder) save.Response {
	t.Helper()

	var resp save.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	return resp
}
