package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Static files (must be first!)
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Home
	mux.HandleFunc("GET /{$}", app.homeHandler)

	// Daily + Weekly habit pages
	mux.HandleFunc("GET /daily", app.habitsHandler)
	mux.HandleFunc("GET /weekly", app.habitsHandler)

	// Progress routes
	mux.HandleFunc("GET /daily/progress", app.progressHandler)
	mux.HandleFunc("GET /weekly/progress", app.progressHandler)

	// Create habit
	mux.HandleFunc("POST /habits/create", app.createHabitHandler)

	// Edit routes - changed to use /habits/edit prefix
	mux.HandleFunc("GET /habits/edit/{frequency}/{id}", app.editHabitHandler)

	// Update route
	mux.HandleFunc("POST /habits/update/{frequency}/{id}", app.updateHabitHandler)

	// Delete habit
	mux.HandleFunc("DELETE /habits/delete/{id}", app.deleteHabitHandler)

	// Log entry for habit completion
	mux.HandleFunc("POST /habits/entries/{id}", app.logEntryHandler)

	return app.session.Enable(app.loggingMiddleware(mux))

}
