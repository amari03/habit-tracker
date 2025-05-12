package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Public routes
	mux.HandleFunc("GET /{$}", app.landingPageHandler)
	mux.HandleFunc("GET /user/signup", app.signupUserForm)
	mux.HandleFunc("POST /user/signup", app.signupUser)
	mux.HandleFunc("GET /user/login", app.loginUserForm)
	mux.HandleFunc("POST /user/login", app.loginUser)

	// Authenticated routes
	mux.Handle("GET /apphome", app.requireAuthentication(http.HandlerFunc(app.homeHandler)))

	// Habit form pages (formerly habitsHandler)
	mux.Handle("GET /daily", app.requireAuthentication(http.HandlerFunc(app.showHabitFormPageHandler)))
	mux.Handle("GET /weekly", app.requireAuthentication(http.HandlerFunc(app.showHabitFormPageHandler)))

	// Habit entries pages (new)
	mux.Handle("GET /daily/entries", app.requireAuthentication(http.HandlerFunc(app.showHabitEntriesPageHandler)))
	mux.Handle("GET /weekly/entries", app.requireAuthentication(http.HandlerFunc(app.showHabitEntriesPageHandler)))

	// Progress routes (called by entries pages)
	mux.Handle("GET /daily/progress", app.requireAuthentication(http.HandlerFunc(app.progressHandler)))
	mux.Handle("GET /weekly/progress", app.requireAuthentication(http.HandlerFunc(app.progressHandler)))

	// Create habit
	mux.Handle("POST /habits/create", app.requireAuthentication(http.HandlerFunc(app.createHabitHandler)))

	// Edit routes
	mux.Handle("GET /habits/edit/{frequency}/{id}", app.requireAuthentication(http.HandlerFunc(app.editHabitHandler)))

	// Update route
	mux.Handle("POST /habits/update/{frequency}/{id}", app.requireAuthentication(http.HandlerFunc(app.updateHabitHandler)))

	// Delete habit - UPDATED ROUTE to include frequency
	mux.Handle("POST /habits/delete/{frequency}/{id}", app.requireAuthentication(http.HandlerFunc(app.deleteHabitHandler)))

	// Log entry for habit completion
	mux.Handle("POST /habits/entries/{id}", app.requireAuthentication(http.HandlerFunc(app.logEntryHandler)))

	// Logout
	mux.Handle("GET /user/logout", app.requireAuthentication(http.HandlerFunc(app.logoutUserHandler)))

	return app.session.Enable(app.noSurf(app.loggingMiddleware(mux)))
}
