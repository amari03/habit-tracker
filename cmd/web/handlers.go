package main

import (
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
		app.logger.Error("failed to render home page", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// dailyHabitsHandler shows daily habits with progress
func (app *application) dailyHabitsHandler(w http.ResponseWriter, r *http.Request) {
	habits, err := app.habits.GetAllByFrequency("daily")
	if err != nil {
		app.logger.Error("failed to fetch daily habits", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Calculate progress (example: percentage of completed habits)
	var completed int
	for _, h := range habits {
		if h.Completed {
			completed++
		}
	}
	progress := 0
	if len(habits) > 0 {
		progress = (completed * 100) / len(habits)
	}

	data := NewTemplateData()
	data.Title = "Daily Habits"
	data.Habits = habits
	data.Progress = progress

	err = app.render(w, http.StatusOK, "daily.tmpl", data)
	if err != nil {
		app.logger.Error("failed to render daily habits", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// createDailyHabitHandler handles new habit creation
func (app *application) createDailyHabitHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.logger.Error("failed to parse form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	habit := &data.Habit{
		Name:       r.PostForm.Get("name"),
		Frequency:  "daily",
		Completed:  false,
	}

	v := validator.NewValidator()
	if habit.Name == "" {
		v.AddError("name", "This field cannot be blank")
	}

	if !v.Valid() {
		data := NewTemplateData()
		data.Title = "Daily Habits"
		data.FormErrors = v.Errors
		data.FormData = map[string]string{
			"name": habit.Name,
		}
		app.render(w, http.StatusUnprocessableEntity, "daily.tmpl", data)
		return
	}

	err = app.habits.Insert(habit)
	if err != nil {
		app.logger.Error("failed to insert habit", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if isHTMXRequest(r) {
		// HTMX response - return just the updated habit list
		habits, _ := app.habits.GetAllByFrequency("daily")
		data := NewTemplateData()
		data.Habits = habits
		app.render(w, http.StatusOK, "partials/habit_list.tmpl", data)
	} else {
		http.Redirect(w, r, "/daily", http.StatusSeeOther)
	}
}

// toggleHabitHandler toggles completed status (HTMX)
func (app *application) toggleHabitHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.logger.Error("invalid habit ID", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	habit, err := app.habits.GetByID(id)
	if err != nil {
		app.logger.Error("habit not found", "error", err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	habit.Completed = !habit.Completed
	err = app.habits.Update(habit)
	if err != nil {
		app.logger.Error("failed to update habit", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return just the updated habit item for HTMX
	data := NewTemplateData()
	data.Habit = habit
	app.render(w, http.StatusOK, "partials/habit_item.tmpl", data)
}

// editHabitFormHandler shows edit form
func (app *application) editHabitFormHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.logger.Error("invalid habit ID", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	habit, err := app.habits.GetByID(id)
	if err != nil {
		app.logger.Error("habit not found", "error", err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := NewTemplateData()
	data.Title = "Edit Habit"
	data.Habit = habit

	err = app.render(w, http.StatusOK, "edit.tmpl", data)
	if err != nil {
		app.logger.Error("failed to render edit form", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// updateHabitHandler processes edit form
func (app *application) updateHabitHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.logger.Error("invalid habit ID", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.logger.Error("failed to parse form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	habit := &data.Habit{
		ID:         id,
		Name:       r.PostForm.Get("name"),
		Frequency:  "daily",
	}

	v := validator.NewValidator()
	if habit.Name == "" {
		v.AddError("name", "This field cannot be blank")
	}

	if !v.Valid() {
		data := NewTemplateData()
		data.Title = "Edit Habit"
		data.Habit = habit
		data.FormErrors = v.Errors
		app.render(w, http.StatusUnprocessableEntity, "edit.tmpl", data)
		return
	}

	err = app.habits.Update(habit)
	if err != nil {
		app.logger.Error("failed to update habit", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/daily", http.StatusSeeOther)
}

// deleteHabitHandler deletes a habit
func (app *application) deleteHabitHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.logger.Error("invalid habit ID", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	err = app.habits.Delete(id)
	if err != nil {
		app.logger.Error("failed to delete habit", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if isHTMXRequest(r) {
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/daily", http.StatusSeeOther)
	}
}

// Helper to check for HTMX requests
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}