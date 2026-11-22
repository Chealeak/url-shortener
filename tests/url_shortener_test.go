package tests

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/config"
	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/random"
)

const host = "localhost:8082"

func TestSave_Success(t *testing.T) {
	user, password := loadAuth(t)
	alias := random.NewRandomString(10)
	target := gofakeit.URL()

	e := newClient(t)

	e.POST("/url").
		WithJSON(save.Request{URL: target, Alias: alias}).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().
		Value("alias").String().IsEqual(alias)

	redirectedToURL, err := api.GetRedirect(endpoint(alias))
	require.NoError(t, err)
	require.Equal(t, target, redirectedToURL)
}

func TestSave_BadRequest(t *testing.T) {
	user, password := loadAuth(t)

	newClient(t).POST("/url").
		WithHeader("Content-Type", "application/json").
		WithBytes([]byte(`{"url":`)).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusBadRequest).
		JSON().Object().
		Value("error").String().IsEqual("Failed to decode request body")
}

func TestSave_Conflict(t *testing.T) {
	user, password := loadAuth(t)
	alias := random.NewRandomString(10)
	body := save.Request{URL: gofakeit.URL(), Alias: alias}

	e := newClient(t)

	e.POST("/url").
		WithJSON(body).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusCreated)

	e.POST("/url").
		WithJSON(body).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusConflict).
		JSON().Object().
		Value("error").String().IsEqual("URL already exists")
}

func TestUpdate_Success(t *testing.T) {
	user, password := loadAuth(t)
	alias := random.NewRandomString(10)
	updatedURL := gofakeit.URL()

	e := newClient(t)

	e.POST("/url").
		WithJSON(save.Request{URL: gofakeit.URL(), Alias: alias}).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusCreated)

	e.PUT("/url").
		WithJSON(map[string]string{
			"alias": alias,
			"url":   updatedURL,
		}).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		Value("countUpdated").Number().IsEqual(1)

	redirectedToURL, err := api.GetRedirect(endpoint(alias))
	require.NoError(t, err)
	require.Equal(t, updatedURL, redirectedToURL)
}

func TestDelete_Success(t *testing.T) {
	user, password := loadAuth(t)
	alias := random.NewRandomString(10)

	e := newClient(t)

	e.POST("/url").
		WithJSON(save.Request{URL: gofakeit.URL(), Alias: alias}).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusCreated)

	e.DELETE("/url/"+alias).
		WithBasicAuth(user, password).
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		Value("countDeleted").Number().IsEqual(1)

	e.GET("/" + alias).
		Expect().
		Status(http.StatusNotFound)
}

func TestAuth_MissingCredentials(t *testing.T) {
	for _, route := range protectedRoutes() {
		t.Run(route.name, func(t *testing.T) {
			req := newClient(t).Request(route.method, route.path)
			if route.body != nil {
				req = req.WithJSON(route.body)
			}

			expectUnauthorized(req.Expect())
		})
	}
}

func TestAuth_WrongCredentials(t *testing.T) {
	user, password := loadAuth(t)

	credentials := []struct {
		name     string
		user     string
		password string
	}{
		{name: "wrong password", user: user, password: password + "-wrong"},
		{name: "wrong user", user: user + "-wrong", password: password},
	}

	for _, route := range protectedRoutes() {
		for _, creds := range credentials {
			t.Run(route.name+"/"+creds.name, func(t *testing.T) {
				req := newClient(t).Request(route.method, route.path).
					WithBasicAuth(creds.user, creds.password)
				if route.body != nil {
					req = req.WithJSON(route.body)
				}

				expectUnauthorized(req.Expect())
			})
		}
	}
}

type protectedRoute struct {
	name   string
	method string
	path   string
	body   any
}

func protectedRoutes() []protectedRoute {
	alias := random.NewRandomString(10)

	return []protectedRoute{
		{
			name:   "save",
			method: http.MethodPost,
			path:   "/url",
			body:   save.Request{URL: gofakeit.URL(), Alias: alias},
		},
		{
			name:   "update",
			method: http.MethodPut,
			path:   "/url",
			body:   map[string]string{"alias": alias, "url": gofakeit.URL()},
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/url/" + alias,
		},
	}
}

func expectUnauthorized(resp *httpexpect.Response) {
	resp.Status(http.StatusUnauthorized)
	resp.Header("WWW-Authenticate").IsEqual(`Basic realm="url-shortener"`)
	resp.Body().IsEmpty()
}

func newClient(t *testing.T) *httpexpect.Expect {
	t.Helper()

	u := url.URL{Scheme: "http", Host: host}
	return httpexpect.Default(t, u.String())
}

func endpoint(alias string) string {
	u := url.URL{Scheme: "http", Host: host, Path: alias}
	return u.String()
}

func loadAuth(t *testing.T) (string, string) {
	t.Helper()

	root := moduleRoot(t)
	require.NoError(t, godotenv.Load(filepath.Join(root, ".env")))

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join(root, "config", "local.yaml")
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(root, configPath)
	}

	var cfg config.Config
	require.NoError(t, cleanenv.ReadConfig(configPath, &cfg))
	require.NotEmpty(t, cfg.User)
	require.NotEmpty(t, cfg.Password)

	return cfg.User, cfg.Password
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod not found")
		dir = parent
	}
}
