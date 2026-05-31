package server

import (
	"io/fs"
	"net/http"
	"strings"

	_ "go-cubemail/docs" // Import registered Swagger spec files

	"github.com/labstack/echo/v5"
	swaggerFiles "github.com/swaggo/files/v2"
	"github.com/swaggo/swag"
)

// registerSwagger registers the Swagger UI natively under the /swagger/* endpoint in Echo v5.
// If enable is false, the Swagger route is omitted from the routing tree entirely.
func registerSwagger(e *echo.Echo, enable bool) {
	if !enable {
		return
	}

	e.GET("/swagger/*", func(c *echo.Context) error {
		p := c.Request().URL.Path
		// Redirect "/swagger" (no trailing slash) to "/swagger/" (with trailing slash)
		// to allow http.FileServer to correctly resolve index.html under the canonical path.
		if p == "/swagger" {
			return c.Redirect(http.StatusMovedPermanently, "/swagger/")
		}

		// Serve the raw registered OpenAPI spec JSON
		if strings.HasSuffix(p, "doc.json") {
			doc, err := swag.ReadDoc()
			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}
			c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
			c.Response().WriteHeader(http.StatusOK)
			_, _ = c.Response().Write([]byte(doc))
			return nil
		}

		// Intercept index.html or root to replace petstore.swagger.io URL with local doc.json
		if p == "/swagger/" || strings.HasSuffix(p, "index.html") {
			content, err := fs.ReadFile(swaggerFiles.FS, "index.html")
			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}
			html := string(content)
			html = strings.ReplaceAll(html, "https://petstore.swagger.io/v2/swagger.json", "doc.json")
			return c.HTML(http.StatusOK, html)
		}

		// Intercept swagger-initializer.js — newer swaggo/files versions moved the URL here
		if strings.HasSuffix(p, "swagger-initializer.js") {
			content, err := fs.ReadFile(swaggerFiles.FS, "swagger-initializer.js")
			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}
			js := strings.ReplaceAll(string(content), "https://petstore.swagger.io/v2/swagger.json", "/swagger/doc.json")
			c.Response().Header().Set("Content-Type", "application/javascript; charset=utf-8")
			return c.String(http.StatusOK, js)
		}

		// Serve static assets from swaggo/files filesystem stripped of the prefix
		handler := http.StripPrefix("/swagger", http.FileServer(http.FS(swaggerFiles.FS)))
		handler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
