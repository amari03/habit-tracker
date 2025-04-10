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
