package main

import (
	"net/http"
)

func (app *application) loggingMiddleware(next http.Handler) http.Handler {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			proto  = r.Proto
			method = r.Method
			uri    = r.URL.RequestURI()
		)

		app.logger.Info("received request", "ip", ip, "protocol", proto, "method", method, "uri", uri)
		next.ServeHTTP(w, r)
		app.logger.Info("Request processed")
	})
	return fn

}

// requireAuthentication is a middleware that ensures a user is logged in.
// If not, it redirects them to the login page.
func (app *application) requireAuthentication(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Use the authenticatedUserID helper. If it returns 0, the user is not authenticated.
		if app.authenticatedUserID(r) == 0 { // Check if user is authenticated
			app.logger.Info("Authentication required, redirecting to login.", "uri", r.URL.RequestURI())

			// Add a flash message to be shown on the login page.
			app.session.Put(r, "flash", "You must be logged in to access this page.")

			// Redirect the user to the login page.
			// http.StatusFound (302) is common for this type of redirect.
			http.Redirect(w, r, "/user/login", http.StatusFound)
			return // Important: Stop processing the request here.
		}

		// If the user *is* authenticated, add a Cache-Control header to prevent
		// browsers/proxies from caching secure pages.
		w.Header().Add("Cache-Control", "no-store")

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	}
	// Wrap the handler function so it satisfies the http.Handler interface.
	return http.HandlerFunc(fn)
}
