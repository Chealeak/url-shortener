package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		respError string
		respAlias string
		mockError error
		status    int
	}{
		{
			name:      "Success",
			alias:     "test_alias",
			url:       "https://google.com",
			respAlias: "test_alias",
			status:    http.StatusCreated,
		},
		{
			name:      "Empty alias",
			alias:     "",
			url:       "https://google.com",
			respAlias: "abc123",
			status:    http.StatusCreated,
		},
		{
			name:      "Empty URL",
			url:       "",
			alias:     "some_alias",
			respError: "Field URL is a required field",
			mockError: service.ErrEmptyURL,
			status:    http.StatusBadRequest,
		},
		{
			name:      "Invalid URL",
			url:       "some invalid URL",
			alias:     "some_alias",
			respError: "Field URL is not a valid URL",
			mockError: service.ErrInvalidURL,
			status:    http.StatusBadRequest,
		},
		{
			name:      "Alias already exists",
			alias:     "test_alias",
			url:       "https://google.com",
			respError: "URL already exists",
			mockError: service.ErrURLExists,
			status:    http.StatusConflict,
		},
		{
			name:      "SaveURL Error",
			alias:     "test_alias",
			url:       "https://google.com",
			respError: "Failed to add URL",
			mockError: errors.New("unexpected error"),
			status:    http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewSaveUrl(t)
			urlSaverMock.On("SaveUrl", mock.Anything, tc.url, tc.alias).
				Return(tc.respAlias, tc.mockError).
				Once()

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock)

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.status, rr.Code)

			var resp save.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			require.Equal(t, tc.respError, resp.Error)
			require.Equal(t, tc.respAlias, resp.Alias)
		})
	}
}
