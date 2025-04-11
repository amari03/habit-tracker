package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/amari03/habit-tracker/internal/data"
	"github.com/amari03/habit-tracker/internal/validator"
)

// homeHandler renders the home page
func (app *application) homeHandler(w http.ResponseWriter, r *http.Request) {
	data := NewTemplateData()
	data.Title = "Home"
	data.Year = time.Now().Year()

	err := app.render(w, http.StatusOK, "home.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// habitsHandler shows habits by frequency (daily/weekly)
func (app *application) habitsHandler(w http.ResponseWriter, r *http.Request) {
	var frequency string

	switch r.URL.Path {
	case "/daily":
		frequency = "daily"
	case "/weekly":
		frequency = "weekly"
	default:
		app.notFound(w)
		return
	}

	habits, err := app.habits.GetAllByFrequency(frequency)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	habitPtrs := make([]*data.Habit, len(habits))
	today := time.Now().Format("2006-01-02")

	for i := range habits {
		habitPtrs[i] = &habits[i]

		entries, err := app.habits.GetEntries(habits[i].ID, time.Now(), time.Now())
		if err == nil && len(entries) > 0 && entries[0].EntryDate.Format("2006-01-02") == today {
			habitPtrs[i].TodayStatus = entries[0].Status
		}
	}

	data := NewTemplateData()
	data.Title = frequency + " Habits"
	data.Habits = habitPtrs
	data.Frequency = frequency

	err = app.render(w, http.StatusOK, frequency+".tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// createHabitHandler handles new habit creation
func (app *application) createHabitHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	habit := &data.Habit{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Frequency:   r.FormValue("frequency"),
		Goal:        r.FormValue("goal"),
	}

	v := validator.NewValidator()
	data.ValidateHabit(v, habit)
	if !v.ValidData() {
		data := NewTemplateData()
		data.FormErrors = v.Errors

		// Convert url.Values to map[string]string
		formData := make(map[string]string)
		for key, values := range r.PostForm {
			if len(values) > 0 {
				formData[key] = values[0] // Take first value
			}
		}
		data.FormData = formData

		err := app.render(w, http.StatusUnprocessableEntity, "daily.tmpl", data)
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.habits.Insert(habit)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := NewTemplateData()
	data.Habit = habit

	err = app.render(w, http.StatusOK, "habit_item.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}

}

// logEntryHandler records a habit completion/skip
func (app *application) logEntryHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	entry := &data.HabitEntry{
		HabitID:   id,
		EntryDate: time.Now(),
		Status:    r.FormValue("status"), // "completed" or "skipped"
		Notes:     r.FormValue("notes"),
	}

	err = app.habits.LogEntry(entry)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// For HTMX requests, return updated habit item
	if isHTMXRequest(r) {
		habit, err := app.habits.GetByID(id)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		entries, err := app.habits.GetEntries(id, time.Now(), time.Now())
		if err == nil && len(entries) > 0 {
			habit.TodayStatus = entries[0].Status
		}

		data := NewTemplateData()
		data.Habit = habit
		app.render(w, http.StatusOK, "partials/habit_item.tmpl", data)
	} else {
		http.Redirect(w, r, r.Header.Get("HX-Current-URL"), http.StatusSeeOther)
	}
}

// editHabitHandler shows the edit form
func (app *application) editHabitHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	habit, err := app.habits.GetByID(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	data := NewTemplateData()
	data.Title = "Edit Habit"
	data.Habit = habit
	data.Frequency = r.PathValue("frequency")

	app.render(w, http.StatusOK, "edit.tmpl", data)
}

// updateHabitHandler processes the edit form
func (app *application) updateHabitHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	habit := &data.Habit{
		ID:          id,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Frequency:   r.FormValue("frequency"),
		Goal:        r.FormValue("goal"),
	}

	v := validator.NewValidator()
	data.ValidateHabit(v, habit)
	if !v.ValidData() {
		data := NewTemplateData()
		data.FormErrors = v.Errors

		// Convert url.Values to map[string]string
		formData := make(map[string]string)
		for key, values := range r.PostForm {
			if len(values) > 0 {
				formData[key] = values[0] // Take first value
			}
		}
		data.FormData = formData

		err := app.render(w, http.StatusUnprocessableEntity, "edit.tmpl", data)
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.habits.Update(habit)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/"+habit.Frequency, http.StatusSeeOther)
}

// deleteHabitHandler removes a habit
func (app *application) deleteHabitHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.habits.Delete(id)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if isHTMXRequest(r) {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/"+r.PathValue("frequency"), http.StatusSeeOther)
	}
}

// progressHandler calculates and returns completion progress
func (app *application) progressHandler(w http.ResponseWriter, r *http.Request) {
	frequency := r.PathValue("frequency")
	if frequency != "daily" && frequency != "weekly" {
		app.notFound(w)
		return
	}

	habits, err := app.habits.GetAllByFrequency(frequency)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// Get completion counts
	var completed, total int
	today := time.Now().Format("2006-01-02") // Format: YYYY-MM-DD

	for _, habit := range habits {
		// Get today's entries
		entries, err := app.habits.GetEntries(habit.ID, time.Now(), time.Now())
		if err == nil && len(entries) > 0 {
			// Check if any entry for today is "completed"
			for _, entry := range entries {
				if entry.EntryDate.Format("2006-01-02") == today && entry.Status == "completed" {
					completed++
					break // Only count one completion per habit
				}
			}
		}
		total++
	}

	// Calculate progress percentage
	progress := 0
	if total > 0 {
		progress = (completed * 100) / total
	}

	// HTMX response
	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="bg-indigo-500 h-4 rounded-full" style="width: ` + strconv.Itoa(progress) + `%"></div>`))
		return
	}

	// Regular response
	data := NewTemplateData()
	data.Progress = progress
	app.render(w, http.StatusOK, "partials/progress_bar.tmpl", data)
}

// Helper to check for HTMX requests
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Helper for server errors
func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// Helper for client errors
func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// Helper for not found errors
func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}
