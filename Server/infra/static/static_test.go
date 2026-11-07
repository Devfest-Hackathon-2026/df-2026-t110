package static

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v5"
)

func newServer(t *testing.T) *echo.Echo {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "public")
	files := map[string]string{
		"index.html":           "index",
		"robot.glb":            "robot",
		"assets/app-abc123.js": "app",
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "secret"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	e.Use(Middleware(dir, "/api"))
	e.GET("/api/me", func(c *echo.Context) error { return c.String(http.StatusOK, "api") })
	return e
}

func TestMiddleware(t *testing.T) {
	t.Parallel()
	e := newServer(t)

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantBody     string
		wantCacheCtl string
	}{
		{name: "hashed asset is cached forever", path: "/assets/app-abc123.js", wantStatus: http.StatusOK, wantBody: "app", wantCacheCtl: cacheImmutable},
		{name: "unhashed public file is revalidated", path: "/robot.glb", wantStatus: http.StatusOK, wantBody: "robot", wantCacheCtl: cacheRevalidate},
		{name: "root serves index", path: "/", wantStatus: http.StatusOK, wantBody: "index", wantCacheCtl: cacheRevalidate},
		{name: "client route falls back to index", path: "/rooms/42", wantStatus: http.StatusOK, wantBody: "index", wantCacheCtl: cacheRevalidate},
		{name: "missing asset is not replaced by index", path: "/assets/missing.js", wantStatus: http.StatusNotFound},
		{name: "API route is left to the API", path: "/api/me", wantStatus: http.StatusOK, wantBody: "api"},
		{name: "unknown API route is not replaced by index", path: "/api/unknown", wantStatus: http.StatusNotFound},
		{name: "path outside the directory falls back to index", path: "/../secret", wantStatus: http.StatusOK, wantBody: "index", wantCacheCtl: cacheRevalidate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCacheCtl {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheCtl)
			}
		})
	}
}
