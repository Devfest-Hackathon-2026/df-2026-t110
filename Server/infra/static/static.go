// Package static serves the built client (SPA) on the same origin as the API.
package static

import (
	"io/fs"
	"os"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const (
	// Vite puts content-hashed files under /assets, so a changed file always gets a new URL.
	assetsPrefix   = "/assets/"
	cacheImmutable = "public, max-age=31536000, immutable"
	// Other files (index.html, public/) keep their URLs across deploys and must be revalidated.
	cacheRevalidate = "no-cache"
)

// Middleware serves files under dir for every path outside apiPrefix. Unknown paths fall back to
// index.html so the client router can handle them, except under /assets where a missing file is a
// real 404 rather than HTML the browser would try to run as a script.
func Middleware(dir, apiPrefix string) echo.MiddlewareFunc {
	files := os.DirFS(dir)
	isAPI := func(c *echo.Context) bool { return strings.HasPrefix(c.Request().URL.Path, apiPrefix) }
	isAsset := func(c *echo.Context) bool { return strings.HasPrefix(c.Request().URL.Path, assetsPrefix) }

	assets := middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       ".",
		Filesystem: files,
		Skipper:    func(c *echo.Context) bool { return !isAsset(c) },
	})
	pages := middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       ".",
		Filesystem: files,
		HTML5:      true,
		Skipper:    func(c *echo.Context) bool { return isAPI(c) || isAsset(c) },
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		serve := assets(pages(next))
		return func(c *echo.Context) error {
			switch {
			case isAPI(c):
			case isAsset(c):
				// Only existing files: a cached 404 would outlive the deploy that adds the file.
				if _, err := fs.Stat(files, strings.TrimPrefix(c.Request().URL.Path, "/")); err == nil {
					c.Response().Header().Set("Cache-Control", cacheImmutable)
				}
			default:
				c.Response().Header().Set("Cache-Control", cacheRevalidate)
			}
			return serve(c)
		}
	}
}
