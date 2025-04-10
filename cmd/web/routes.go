package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Static files (must be first!)
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Home
	mux.HandleFunc("GET /", app.homeHandler)

	// Daily + Weekly habit pages
	mux.HandleFunc("GET /daily", app.habitsHandler)
	mux.HandleFunc("GET /weekly", app.habitsHandler)

	// Progress routes
	mux.HandleFunc("GET /daily/progress", app.progressHandler)
	mux.HandleFunc("GET /weekly/progress", app.progressHandler)

	// Edit pages (no wildcards to avoid conflict)
	mux.HandleFunc("GET /daily/edit/{id}", app.editHabitHandler)
	mux.HandleFunc("GET /weekly/edit/{id}", app.editHabitHandler)

	// Create habit
	mux.HandleFunc("POST /habits/create", app.createHabitHandler)

	// Update habit (clear route)
	mux.HandleFunc("POST /habits/{id}/update", app.updateHabitHandler)

	// Delete habit (same here)
	mux.HandleFunc("DELETE /habits/{id}/delete", app.deleteHabitHandler)

	// Log entry for habit completion
	mux.HandleFunc("POST /habits/{id}/entries", app.logEntryHandler)

	return app.loggingMiddleware(mux)
}
