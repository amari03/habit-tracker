package main

import (
	"errors"
	//"fmt"
	//"html/template"
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

// landingHandler renders the landing page
func (app *application) landingPageHandler(w http.ResponseWriter, r *http.Request) {
	// If the user is already authenticated, maybe redirect them to their dashboard/home
	// For now, we'll always show the landing page.
	// if app.isAuthenticated(r) { // You'll need an isAuthenticated helper
	// http.Redirect(w, r, "/daily", http.StatusSeeOther)
	// return
	// }

	data := NewTemplateData()
	data.Title = "Welcome" // Title for the landing page
	// data.Flash = app.session.PopString(r, "flash") // If landing page needs to show flash messages

	err := app.render(w, http.StatusOK, "landing.tmpl", data)
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
	app.logger.Info("Create habit request received", "method", r.Method, "url", r.URL)

	err := r.ParseForm()
	if err != nil {
		app.logger.Error("Failed to parse form", "error", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	habit := &data.Habit{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Frequency:   r.FormValue("frequency"),
		Goal:        r.FormValue("goal"),
	}

	app.logger.Info("Habit data received",
		"title", habit.Title,
		"description", habit.Description,
		"frequency", habit.Frequency,
		"goal", habit.Goal)

	v := validator.NewValidator()
	data.ValidateHabit(v, habit)

	//  Validation error
	if !v.ValidData() {
		form := NewTemplateData()
		form.FormErrors = v.Errors
		form.FormData = map[string]string{
			"title":       habit.Title,
			"description": habit.Description,
			"goal":        habit.Goal,
		}
		form.Frequency = habit.Frequency
		form.Habits = []*data.Habit{} // avoid nil panic if rendered

		// If HTMX, render just the form partial
		if r.Header.Get("HX-Request") == "true" {
			err := app.renderPartial(w, http.StatusUnprocessableEntity, "partials/habit_form.tmpl", form)
			if err != nil {
				app.serverError(w, r, err)
			}
		} else {
			// fallback full render
			err := app.render(w, http.StatusUnprocessableEntity, "daily.tmpl", form)
			if err != nil {
				app.serverError(w, r, err)
			}
		}
		return
	}

	// Insert the habit
	err = app.habits.Insert(habit)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// Get all habits to refresh the list
	habits, err := app.habits.GetAllByFrequency(habit.Frequency)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// Convert to pointers for template
	habitPtrs := make([]*data.Habit, len(habits))
	today := time.Now().Format("2006-01-02")

	for i := range habits {
		habitPtrs[i] = &habits[i]

		// Get today's status for each habit
		entries, err := app.habits.GetEntries(habits[i].ID, time.Now(), time.Now())
		if err == nil && len(entries) > 0 && entries[0].EntryDate.Format("2006-01-02") == today {
			habitPtrs[i].TodayStatus = entries[0].Status
		}
	}

	// Return a fresh form and update the habit list
	if isHTMXRequest(r) {
		// Create a fresh form container with a new form
		formData := NewTemplateData()
		formData.FormData = map[string]string{
			"frequency": habit.Frequency,
		}

		// Return the form container with a fresh form
		w.Header().Set("HX-Trigger", `{"refreshHabitsList": "#habits-list"}`)
		err = app.renderPartial(w, http.StatusOK, "partials/habit_form.tmpl", formData)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
	} else {
		// For non-HTMX requests, redirect to the habits page
		http.Redirect(w, r, "/"+habit.Frequency, http.StatusSeeOther)
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

	// For HTMX requests, trigger a refresh of the habits list
	if isHTMXRequest(r) {
		// Set the HX-Trigger header to refresh the habits list
		w.Header().Set("HX-Trigger", `{"refreshHabitsList": "#habits-list"}`)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, r.Header.Get("HX-Current-URL"), http.StatusSeeOther)
	}
}

// editHabitHandler shows the edit form
func (app *application) editHabitHandler(w http.ResponseWriter, r *http.Request) {
	// Get both parameters from the path
	frequency := r.PathValue("frequency")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate frequency
	if frequency != "daily" && frequency != "weekly" {
		app.notFound(w)
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
	data.Frequency = frequency
	data.PermittedFrequencies = []string{"daily", "weekly"}

	err = app.render(w, http.StatusOK, "edit.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// updateHabitHandler processes the edit form
func (app *application) updateHabitHandler(w http.ResponseWriter, r *http.Request) {
	// Get both parameters from the path
	frequency := r.PathValue("frequency")
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
		Frequency:   r.FormValue("frequency"), // Use the form value
		Goal:        r.FormValue("goal"),
	}

	v := validator.NewValidator()
	data.ValidateHabit(v, habit)
	if !v.ValidData() {
		data := NewTemplateData()
		data.FormErrors = v.Errors
		data.Habit = habit
		data.Frequency = frequency // Use the path parameter here
		data.PermittedFrequencies = []string{"daily", "weekly"}

		formData := make(map[string]string)
		for key, values := range r.PostForm {
			if len(values) > 0 {
				formData[key] = values[0]
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

	// Use the frequency parameter for redirect
	http.Redirect(w, r, "/"+frequency, http.StatusSeeOther)
}

// deleteHabitHandler removes a habit
func (app *application) deleteHabitHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.logger.Error("Invalid habit ID", "error", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	app.logger.Info("Deleting habit", "id", id)

	err = app.habits.Delete(id)
	if err != nil {
		app.logger.Error("Failed to delete habit", "error", err)
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
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

func (app *application) signupUserForm(w http.ResponseWriter, r *http.Request) {
	data := NewTemplateData()
	data.Title = "Sign Up"

	// signup.tmpl is now a standalone page
	err := app.render(w, http.StatusOK, "signup.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}

}

func (app *application) signupUser(w http.ResponseWriter, r *http.Request) {
	// 1. Parse form data
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// 2. Extract values
	name := r.PostForm.Get("name")
	email := r.PostForm.Get("email")
	passwordInput := r.PostForm.Get("password") // Plaintext password

	// 3. Initialize Validator
	v := validator.NewValidator()

	// 4. Create User struct (without password yet)
	// The password field is now handled by the data.password type
	user := &data.User{
		Name:  name,
		Email: email,
		// Active will be set later based on example logic
	}

	// 5. Validate name and email using your existing function
	data.ValidateUser(v, user) // Directly call the validation function

	// 6. Validate plaintext password (rules defined in the handler)
	v.Check(validator.NotBlank(passwordInput), "password", "Password must be provided")
	v.Check(validator.MinLength(passwordInput, 8), "password", "Password must be at least 8 characters long")
	v.Check(validator.MaxLength(passwordInput, 72), "password", "Password must not be more than 72 characters")

	// 7. Check if validation failed
	if !v.ValidData() {
		app.logger.Info("Signup validation failed", "errors", v.Errors)
		// Prepare data for re-rendering the form
		formData := NewTemplateData() // Use your existing helper
		formData.Title = "Sign Up - Error"
		formData.FormErrors = v.Errors
		// Repopulate form data (excluding password)
		formData.FormData = map[string]string{
			"name":  name,
			"email": email,
		}

		// Render the signup page again with errors
		errRender := app.render(w, http.StatusUnprocessableEntity, "signup.tmpl", formData)
		if errRender != nil {
			app.serverError(w, r, errRender)
		}
		return
	}

	// --- Validation Passed ---

	// 8. Set and Hash the password using the User struct's method
	err = user.Password.Set(passwordInput)
	if err != nil {
		app.serverError(w, r, err) // Handle potential bcrypt errors
		return
	}

	// 9. Set user as active (based on example logic)
	user.Active = true

	// 10. Insert the user data into the database
	err = app.users.Insert(user) // This now sends the hashed password correctly
	if err != nil {
		// Check specifically for the duplicate email error returned by Insert
		if errors.Is(err, data.ErrDuplicateEmail) {
			v.AddError("email", "Email address is already registered") // Add error *to the validator*

			app.logger.Info("Signup failed due to duplicate email", "email", email)
			// Re-render the form with the duplicate email error added to FormErrors
			formData := NewTemplateData()
			formData.Title = "Sign Up - Error"
			formData.FormErrors = v.Errors // Pass the updated validator errors
			formData.FormData = map[string]string{
				"name":  name,
				"email": email,
			}
			errRender := app.render(w, http.StatusUnprocessableEntity, "signup.tmpl", formData)
			if errRender != nil {
				app.serverError(w, r, errRender)
			}
		} else {
			// Handle any other database insertion errors
			app.serverError(w, r, err)
		}
		return // Stop processing on error
	}

	// 11. Success! User was created.
	app.logger.Info("User signed up successfully", "userID", user.ID, "email", user.Email)

	// Add a flash message to the session.
	app.session.Put(r, "flash", "Your signup was successful! Please log in.")

	// 12. Redirect the user to the login page.
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) loginUserForm(w http.ResponseWriter, r *http.Request) {
	data := NewTemplateData()
	data.Title = "Login"
	// Retrieve and remove the flash message from the session.
	data.Flash = app.session.PopString(r, "flash")

	// Note: login.tmpl is now parsed as a standalone page (no nav bar)
	err := app.render(w, http.StatusOK, "login.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) loginUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	email := r.PostForm.Get("email")
	passwordInput := r.PostForm.Get("password")

	v := validator.NewValidator() // Create a new validator instance

	// Basic checks for blank fields.
	// The error key "generic" will be used by login.tmpl to display a general message.
	if !validator.NotBlank(email) || !validator.NotBlank(passwordInput) {
		v.AddError("generic", "Both email and password must be provided.")
	}

	// If there are any validation errors (e.g., blank fields)
	if !v.ValidData() {
		data := NewTemplateData()
		data.Title = "Login - Error"
		data.FormData = map[string]string{"email": email} // Repopulate email
		data.FormErrors = v.Errors                        // Pass all validation errors

		errRender := app.render(w, http.StatusUnprocessableEntity, "login.tmpl", data)
		if errRender != nil {
			app.serverError(w, r, errRender)
		}
		return
	}

	// Attempt to authenticate the user
	id, err := app.users.Authenticate(email, passwordInput)
	if err != nil {
		if errors.Is(err, data.ErrInvalidCredentials) {
			// Add a generic error for invalid credentials
			v.AddError("generic", "Invalid email or password.")

			data := NewTemplateData()
			data.Title = "Login - Error"
			data.FormData = map[string]string{"email": email} // Repopulate email
			data.FormErrors = v.Errors                        // Pass the updated errors

			errRender := app.render(w, http.StatusUnprocessableEntity, "login.tmpl", data)
			if errRender != nil {
				app.serverError(w, r, errRender)
			}
		} else {
			// Any other error from Authenticate is a server error
			app.serverError(w, r, err)
		}
		return
	}

	// Authentication successful
	app.session.Put(r, "authenticatedUserID", id) // Store user ID in session
	app.session.Put(r, "flash", "You have been logged in successfully!")

	// Redirect to a relevant page, e.g., daily habits or home
	http.Redirect(w, r, "/apphome", http.StatusSeeOther)
}

// logoutUserHandler handles user logout
func (app *application) logoutUserHandler(w http.ResponseWriter, r *http.Request) {

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
