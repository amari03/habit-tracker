package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	
	// Static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// Home route
	mux.HandleFunc("GET /", app.homeHandler)

	// Unified habit routes
	mux.HandleFunc("GET /{frequency}", app.habitsHandler)                // daily/weekly
	mux.HandleFunc("POST /habits/create", app.createHabitHandler)       // create new
	mux.HandleFunc("POST /habits/{id}/entries", app.logEntryHandler)    // log completion
	mux.HandleFunc("GET /{frequency}/edit/{id}", app.editHabitHandler)  // edit form
	mux.HandleFunc("POST /habits/update/{id}", app.updateHabitHandler)  // update
	mux.HandleFunc("DELETE /habits/delete/{id}", app.deleteHabitHandler) // delete
	mux.HandleFunc("GET /{frequency}/progress", app.progressHandler)    // progress bar

	return app.loggingMiddleware(mux)
}