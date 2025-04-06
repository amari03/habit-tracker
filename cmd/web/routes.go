package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	
	// Static files
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))

	// Home route
	mux.HandleFunc("GET /", app.homeHandler)

	// Daily Habits routes
	mux.HandleFunc("GET /daily", app.dailyHabitsHandler)
	mux.HandleFunc("POST /daily/create", app.createDailyHabitHandler)
	mux.HandleFunc("POST /daily/toggle/{id}", app.toggleHabitHandler)
	mux.HandleFunc("GET /daily/edit/{id}", app.editHabitFormHandler)
	mux.HandleFunc("POST /daily/update/{id}", app.updateHabitHandler)
	mux.HandleFunc("DELETE /daily/delete/{id}", app.deleteHabitHandler)
	mux.HandleFunc("GET /daily/progress", app.progressHandler)

	// Weekly Habits routes (mirror daily structure)
	mux.HandleFunc("GET /weekly", app.weeklyHabitsHandler)
	mux.HandleFunc("POST /weekly/create", app.createWeeklyHabitHandler)
	mux.HandleFunc("POST /weekly/toggle/{id}", app.toggleHabitHandler)
	mux.HandleFunc("GET /weekly/edit/{id}", app.editHabitFormHandler)
	mux.HandleFunc("POST /weekly/update/{id}", app.updateHabitHandler)
	mux.HandleFunc("DELETE /weekly/delete/{id}", app.deleteHabitHandler)
	mux.HandleFunc("GET /weekly/progress", app.progressHandler)

	// Add middleware chain
	return app.loggingMiddleware(mux)
}