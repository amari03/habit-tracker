package main

import (
	"log/slog"
	"net/http"
	"time"
	"crypto/tls"
)

func (app *application) serve() error {
	// Load TLS configuration
	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
	}


	srv := &http.Server{
		Addr:         *app.addr,
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
	}
	app.logger.Info("starting server", "addr", srv.Addr)
	return srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
}
