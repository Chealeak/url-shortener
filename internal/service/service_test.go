package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"url-shortener/internal/service"
	"url-shortener/internal/storage"
)

type savedURL struct {
	ctx   context.Context
	url   string
	alias string
}

type fakeStorage struct {
	calls []savedURL
	errs  []error
}

func (f *fakeStorage) UpdateUrl(ctx context.Context, alias string, newUrl string) (int64, error) {
	f.calls = append(f.calls, savedURL{ctx: ctx, url: newUrl, alias: alias})

	if len(f.errs) == 0 {
		return 1, nil
	}

	err := f.errs[0]
	f.errs = f.errs[1:]
	if err != nil {
		return 0, err
	}

	return 1, nil
}

func (f *fakeStorage) SaveUrl(ctx context.Context, urlToSave string, alias string) (int64, error) {
	f.calls = append(f.calls, savedURL{ctx: ctx, url: urlToSave, alias: alias})

	if len(f.errs) == 0 {
		return int64(len(f.calls)), nil
	}

	err := f.errs[0]
	f.errs = f.errs[1:]
	if err != nil {
		return 0, err
	}

	return int64(len(f.calls)), nil
}

func TestUpdateUrl(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name       string
		alias      string
		url        string
		storageErr []error
		wantErr    error
		wantCount  int64
		wantCalls  int
	}{
		{
			name:    "empty alias",
			alias:   "",
			url:     "https://google.com",
			wantErr: service.ErrEmptyAlias,
		},
		{
			name:    "invalid url",
			alias:   "google",
			url:     "not a url",
			wantErr: service.ErrInvalidURL,
		},
		{
			name:      "updates url",
			alias:     "google",
			url:       "https://google.com",
			wantCount: 1,
			wantCalls: 1,
		},
		{
			name:       "alias not found",
			alias:      "missing",
			url:        "https://google.com",
			storageErr: []error{fmt.Errorf("update: %w", storage.ErrUrlNotFound)},
			wantErr:    service.ErrURLNotFound,
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeStorage{errs: tt.storageErr}
			svc := service.New(store)

			count, err := svc.UpdateUrl(ctx, tt.alias, tt.url)

			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantCount, count)
			}

			require.Len(t, store.calls, tt.wantCalls)
		})
	}
}

func TestSaveUrl(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")

	tests := []struct {
		name       string
		url        string
		alias      string
		storageErr []error
		wantErr    error
		wantAlias  string
		wantCalls  int
	}{
		{
			name:    "empty url",
			url:     "",
			alias:   "alias",
			wantErr: service.ErrEmptyURL,
		},
		{
			name:    "invalid url",
			url:     "not a url",
			alias:   "alias",
			wantErr: service.ErrInvalidURL,
		},
		{
			name:      "saves provided alias",
			url:       "https://google.com",
			alias:     "google",
			wantAlias: "google",
			wantCalls: 1,
		},
		{
			name:      "generates alias",
			url:       "https://google.com",
			wantCalls: 1,
		},
		{
			name:       "provided alias already exists",
			url:        "https://google.com",
			alias:      "google",
			storageErr: []error{fmt.Errorf("insert: %w", storage.ErrUrlExists)},
			wantErr:    service.ErrURLExists,
			wantCalls:  1,
		},
		{
			name: "retries generated alias on conflict",
			url:  "https://google.com",
			storageErr: []error{
				fmt.Errorf("insert: %w", storage.ErrUrlExists),
				nil,
			},
			wantCalls: 2,
		},
		{
			name:       "returns storage error",
			url:        "https://google.com",
			alias:      "google",
			storageErr: []error{errors.New("db is down")},
			wantErr:    errors.New("save url: db is down"),
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeStorage{errs: tt.storageErr}
			svc := service.New(store)

			alias, err := svc.SaveUrl(ctx, tt.url, tt.alias)

			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}

			if tt.wantAlias != "" {
				require.Equal(t, tt.wantAlias, alias)
			}
			if tt.wantErr == nil && tt.alias == "" {
				require.Len(t, alias, 6)
			}

			require.Len(t, store.calls, tt.wantCalls)
			for _, call := range store.calls {
				require.Equal(t, "request", call.ctx.Value(ctxKey{}))
				require.Equal(t, tt.url, call.url)
				require.NotEmpty(t, call.alias)
			}
		})
	}
}
