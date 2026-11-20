package update_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/update"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service"
)

type updateStub struct {
	called bool
	count  int64
	err    error
}

func (s *updateStub) UpdateUrl(context.Context, string, string) (int64, error) {
	s.called = true
	return s.count, s.err
}

func TestUpdate_Success(t *testing.T) {
	stub := &updateStub{count: 1}

	rr := callUpdate(t, stub, `{"alias":"google","url":"https://google.com"}`)

	require.True(t, stub.called)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp update.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Empty(t, resp.Error)
	require.Equal(t, int64(1), resp.CountUpdated)
}

func TestUpdate_BadRequest(t *testing.T) {
	stub := &updateStub{count: 1}

	rr := callUpdate(t, stub, `{"alias":`)

	require.False(t, stub.called)
	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp update.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "Failed to decode request body", resp.Error)
}

func TestUpdate_NotFound(t *testing.T) {
	stub := &updateStub{err: service.ErrAliasNotFound}

	rr := callUpdate(t, stub, `{"alias":"missing","url":"https://google.com"}`)

	require.Equal(t, http.StatusNotFound, rr.Code)

	var resp update.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "Alias not found", resp.Error)
}

func TestUpdate_InternalError(t *testing.T) {
	stub := &updateStub{err: errors.New("db is down")}

	rr := callUpdate(t, stub, `{"alias":"google","url":"https://google.com"}`)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp update.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "Failed to update URL", resp.Error)
}

func callUpdate(t *testing.T, updater update.UpdateUrl, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := update.New(slogdiscard.NewDiscardLogger(), updater)
	req := httptest.NewRequest(http.MethodPut, "/url", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}
