package middleware

import "github.com/labstack/echo/v5"

// SecurityHeaders sets defensive HTTP response headers on every request:
// X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy, and CSP.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			// style-src 'unsafe-inline' required for runtime CSS injection (Tailwind vars, accent picker)
			// img-src https: allows external images in emails and the composer (newsletters, avatars, CDN assets)
			// img-src data:/blob:/cid: covers inline attachments and base64-encoded email images
			h.Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self'; "+
					"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
					"font-src 'self' https://fonts.gstatic.com data:; "+
					"img-src 'self' https: data: cid: blob:; "+
					"connect-src 'self'; "+
					"frame-src 'self' blob:; "+
					"frame-ancestors 'none'; "+
					"form-action 'self'",
			)
			return next(c)
		}
	}
}
