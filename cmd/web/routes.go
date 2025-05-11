package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Static files (must be first!)
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// --- Publicly accessible routes ---
	mux.HandleFunc("GET /{$}", app.landingPageHandler)     // Landing page
	mux.HandleFunc("GET /user/signup", app.signupUserForm) // Signup form
	mux.HandleFunc("POST /user/signup", app.signupUser)    // Signup action
	mux.HandleFunc("GET /user/login", app.loginUserForm)   // Login form
	mux.HandleFunc("POST /user/login", app.loginUser)      // Login action

	// --- Authenticated routes ---
	// We will create a new ServeMux for protected routes and apply middleware to it.
	// This is cleaner than wrapping each handler individually IF all routes under a certain path prefix are protected.
	// However, with diverse paths, individual wrapping might be necessary.
	// For this setup, since paths are diverse, we'll wrap handlers.

	// Authenticated user's "home" page
	mux.Handle("GET /apphome", app.requireAuthentication(http.HandlerFunc(app.homeHandler)))

	// Daily + Weekly habit pages
	mux.Handle("GET /daily", app.requireAuthentication(http.HandlerFunc(app.habitsHandler)))
	mux.Handle("GET /weekly", app.requireAuthentication(http.HandlerFunc(app.habitsHandler)))

	// Progress routes
	mux.Handle("GET /daily/progress", app.requireAuthentication(http.HandlerFunc(app.progressHandler)))
	mux.Handle("GET /weekly/progress", app.requireAuthentication(http.HandlerFunc(app.progressHandler)))

	// Create habit
	mux.Handle("POST /habits/create", app.requireAuthentication(http.HandlerFunc(app.createHabitHandler)))

	// Edit routes
	mux.Handle("GET /habits/edit/{frequency}/{id}", app.requireAuthentication(http.HandlerFunc(app.editHabitHandler)))

	// Update route
	mux.Handle("POST /habits/update/{frequency}/{id}", app.requireAuthentication(http.HandlerFunc(app.updateHabitHandler)))

	// Delete habit
	mux.Handle("POST /habits/delete/{id}", app.requireAuthentication(http.HandlerFunc(app.deleteHabitHandler)))

	// Log entry for habit completion
	mux.Handle("POST /habits/entries/{id}", app.requireAuthentication(http.HandlerFunc(app.logEntryHandler)))

	// Logout route (typically requires authentication to be meaningful, though the handler itself doesn't enforce it)
	mux.Handle("GET /user/logout", app.requireAuthentication(http.HandlerFunc(app.logoutUserHandler)))

	// The session middleware should wrap everything, then logging, then the mux with its routes.
	// The requireAuthentication middleware is applied to specific handlers.
	return app.session.Enable(app.noSurf(app.loggingMiddleware(mux)))
}
