package delete_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service"
)

type deleteStub struct {
	alias  string
	called bool
	count  int64
	err    error
}

func (s *deleteStub) DeleteUrl(_ context.Context, alias string) (int64, error) {
	s.called = true
	s.alias = alias
	return s.count, s.err
}

func TestDelete_Success(t *testing.T) {
	stub := &deleteStub{count: 1}

	rr := callDelete(t, stub, "/google")

	require.True(t, stub.called)
	require.Equal(t, "google", stub.alias)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp delete.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Empty(t, resp.Error)
	require.Equal(t, int64(1), resp.CountDeleted)
}

func TestDelete_NotFound(t *testing.T) {
	stub := &deleteStub{err: service.ErrAliasNotFound}

	rr := callDelete(t, stub, "/missing")

	require.Equal(t, http.StatusNotFound, rr.Code)

	var resp delete.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "Alias not found", resp.Error)
}

func TestDelete_InternalError(t *testing.T) {
	stub := &deleteStub{err: errors.New("db is down")}

	rr := callDelete(t, stub, "/google")

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp delete.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, "Failed to delete URL", resp.Error)
}

func callDelete(t *testing.T, remover delete.DeleteUrl, target string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Delete("/{alias}", delete.New(slogdiscard.NewDiscardLogger(), remover))

	req := httptest.NewRequest(http.MethodDelete, target, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}
