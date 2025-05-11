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

// authenticatedUserID returns the ID of the currently authenticated user from the session.
// Returns 0 if no user is authenticated or if the ID is invalid.
func (app *application) authenticatedUserID(r *http.Request) int64 {
	id, ok := app.session.Get(r, "authenticatedUserID").(int64)
	if !ok {
		return 0 // Or some other indicator for "not authenticated"
	}
	return id
}

// homeHandler renders the home page
func (app *application) homeHandler(w http.ResponseWriter, r *http.Request) {
	// You might still want the userID if this page needs to display user-specific info
	// userID := app.authenticatedUserID(r)
	// app.logger.Info("Home page for user", "userID", userID)

	data := NewTemplateData()
	data.Title = "Home"

	data.IsAuthenticated = true // <<< SET IsAuthenticated

	err := app.render(w, r, http.StatusOK, "home.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// landingHandler renders the landing page
func (app *application) landingPageHandler(w http.ResponseWriter, r *http.Request) {

	data := NewTemplateData()
	data.Title = "Welcome" // Title for the landing page
	// data.Flash = app.session.PopString(r, "flash") // If landing page needs to show flash messages

	err := app.render(w, r, http.StatusOK, "landing.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// habitsHandler shows habits by frequency (daily/weekly) for the authenticated user
func (app *application) habitsHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	var frequency string
	switch r.URL.Path {
	case "/daily":
		frequency = "daily"
	case "/weekly":
		frequency = "weekly"
	default:
		app.notFound(w) // Or redirect to /apphome or /user/login
		return
	}

	// Fetch habits for the specific user and frequency
	habits, err := app.habits.GetAllByFrequency(userID, frequency) // <<< PASS userID
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	habitPtrs := make([]*data.Habit, len(habits))
	today := time.Now().Format("2006-01-02")

	for i := range habits {
		habitPtrs[i] = &habits[i]
		// Fetching entries remains the same, as it's by habit.ID
		entries, err := app.habits.GetEntries(habits[i].ID, time.Now(), time.Now())
		if err == nil && len(entries) > 0 && entries[0].EntryDate.Format("2006-01-02") == today {
			habitPtrs[i].TodayStatus = entries[0].Status
		}
	}

	templatePageData := NewTemplateData()
	templatePageData.Title = frequency + " Habits"
	templatePageData.Habits = habitPtrs
	templatePageData.Frequency = frequency
	templatePageData.IsAuthenticated = true // User is authenticated to see this page
	templatePageData.Flash = app.session.PopString(r, "flash")

	err = app.render(w, r, http.StatusOK, frequency+".tmpl", templatePageData)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// createHabitHandler handles new habit creation
func (app *application) createHabitHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		// If you have a middleware for authentication, this might be redundant,
		// but it's a good safeguard.
		app.clientError(w, http.StatusUnauthorized) // Or redirect to login
		return
	}

	app.logger.Info("Create habit request received", "method", r.Method, "url", r.URL, "userID", userID)

	err := r.ParseForm()
	if err != nil {
		app.logger.Error("Failed to parse form", "error", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	habit := &data.Habit{
		UserID:      userID, // <<< SET UserID HERE
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Frequency:   r.FormValue("frequency"),
		Goal:        r.FormValue("goal"),
	}

	app.logger.Info("Habit data received",
		"userID", habit.UserID,
		"title", habit.Title,
		"description", habit.Description,
		"frequency", habit.Frequency,
		"goal", habit.Goal)

	v := validator.NewValidator()
	data.ValidateHabit(v, habit) // ValidateHabit itself doesn't change for UserID

	if !v.ValidData() {
		// ... (error handling for validation remains largely the same)
		// Ensure you are creating NewTemplateData and passing it correctly
		formTemplateData := NewTemplateData() // Use your constructor
		formTemplateData.FormErrors = v.Errors
		formTemplateData.FormData = map[string]string{
			"title":       habit.Title,
			"description": habit.Description,
			"goal":        habit.Goal,
		}
		formTemplateData.Frequency = habit.Frequency
		formTemplateData.IsAuthenticated = (userID != 0) // Add this if base template needs it for nav

		if isHTMXRequest(r) {
			err := app.renderPartial(w, r, http.StatusUnprocessableEntity, "partials/habit_form.tmpl", formTemplateData)
			if err != nil {
				app.serverError(w, r, err)
			}
		} else {
			// Fallback full render - you might need to decide which page to render here
			// e.g., if frequency is known, render that page.
			// For simplicity, let's assume daily for now if it's a full page error.
			formTemplateData.Title = habit.Frequency + " Habits - Error"
			habits, _ := app.habits.GetAllByFrequency(userID, habit.Frequency) // Fetch habits for the current user to display the page correctly
			habitPtrs := make([]*data.Habit, len(habits))
			for i := range habits {
				habitPtrs[i] = &habits[i]
			}
			formTemplateData.Habits = habitPtrs
			err := app.render(w, r, http.StatusUnprocessableEntity, habit.Frequency+".tmpl", formTemplateData)
			if err != nil {
				app.serverError(w, r, err)
			}
		}
		return
	}

	err = app.habits.Insert(habit) // Insert now includes UserID
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// On success, refresh the list or redirect
	// The logic for HTMX refresh or full redirect can remain similar,
	// but ensure the refreshed list also considers the userID.

	// If HTMX, return a fresh form and trigger list refresh
	if isHTMXRequest(r) {
		freshFormData := NewTemplateData()
		freshFormData.FormData = map[string]string{
			"frequency": habit.Frequency, // Preserve frequency for the new form
		}
		// freshFormData.IsAuthenticated = true

		w.Header().Set("HX-Trigger", `{"refreshHabitsList": "#habits-list"}`)
		err = app.renderPartial(w, r, http.StatusOK, "partials/habit_form.tmpl", freshFormData)
		if err != nil {
			app.serverError(w, r, err)
		}
	} else {
		http.Redirect(w, r, "/"+habit.Frequency, http.StatusSeeOther)
	}
}

// logEntryHandler records a habit completion/skip
func (app *application) logEntryHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	habitID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// <<< AUTHORIZATION CHECK: Verify the habit belongs to the current user >>>
	habit, err := app.habits.GetByID(habitID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	if habit.UserID != userID {
		app.logger.Warn("Forbidden access attempt to log entry for habit", "habitID", habitID, "requesterUserID", userID)
		app.notFound(w) // Or app.clientError(w, http.StatusForbidden)
		return
	}

	// Now that we've confirmed the habit belongs to the user, proceed
	entry := &data.HabitEntry{
		HabitID:   habitID, // Use the validated habitID
		EntryDate: time.Now(),
		Status:    r.FormValue("status"), // "completed" or "skipped"
		Notes:     r.FormValue("notes"),
	}

	err = app.habits.LogEntry(entry) // This method is on HabitModel but operates on habit_entries table
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if isHTMXRequest(r) {
		w.Header().Set("HX-Trigger", `{"refreshHabitsList": "#habits-list"}`)
		w.WriteHeader(http.StatusOK)
	} else {
		// For non-HTMX, redirect back to the page they were on.
		// HX-Current-URL is an HTMX header. For general cases, Referer might be an option,
		// or redirect to a known page like /daily or /weekly based on habit.Frequency.
		http.Redirect(w, r, "/"+habit.Frequency, http.StatusSeeOther)
	}
}

// editHabitHandler shows the edit form if the habit belongs to the user
func (app *application) editHabitHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	frequencyPathValue := r.PathValue("frequency") // Renamed to avoid conflict with templateData.Frequency
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if frequencyPathValue != "daily" && frequencyPathValue != "weekly" {
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

	// <<< AUTHORIZATION CHECK >>>
	if habit.UserID != userID {
		app.logger.Warn("Forbidden access attempt to edit habit", "habitID", habit.ID, "habitUserID", habit.UserID, "requesterUserID", userID)
		app.notFound(w) // Treat as not found to avoid leaking info
		return
	}

	editTemplateData := NewTemplateData()
	editTemplateData.Title = "Edit Habit"
	editTemplateData.Habit = habit
	editTemplateData.Frequency = frequencyPathValue // Use the path value for the form's context
	editTemplateData.PermittedFrequencies = []string{"daily", "weekly"}
	editTemplateData.IsAuthenticated = true

	// Repopulate FormData if coming from a failed update attempt (though less common for GET)
	// Or, you might want to populate FormData from the habit itself for the initial edit form display.
	editTemplateData.FormData = map[string]string{
		"title":       habit.Title,
		"description": habit.Description,
		"frequency":   habit.Frequency, // current frequency of the habit
		"goal":        habit.Goal,
	}

	err = app.render(w, r, http.StatusOK, "edit.tmpl", editTemplateData)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// updateHabitHandler processes the edit form if the habit belongs to the user
func (app *application) updateHabitHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	frequencyPathValue := r.PathValue("frequency") // Original frequency from path
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

	// Create habit struct with form data
	habitToUpdate := &data.Habit{
		ID:          id,
		UserID:      userID, // <<< SET UserID for the update operation
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Frequency:   r.FormValue("frequency"), // New frequency from form
		Goal:        r.FormValue("goal"),
	}

	// First, verify ownership by fetching the original habit (optional but safer before validation)
	// Alternatively, the Model.Update includes UserID in its WHERE clause.
	// For thoroughness, let's ensure the habit exists and belongs to the user before attempting to validate/update.
	originalHabit, err := app.habits.GetByID(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}
	if originalHabit.UserID != userID {
		app.logger.Warn("Forbidden access attempt to update habit", "habitID", id, "requesterUserID", userID)
		app.notFound(w) // Or app.clientError(w, http.StatusForbidden)
		return
	}

	v := validator.NewValidator()
	data.ValidateHabit(v, habitToUpdate) // Validate the new data
	if !v.ValidData() {
		errorTemplateData := NewTemplateData()
		errorTemplateData.FormErrors = v.Errors
		errorTemplateData.Habit = habitToUpdate // Pass the data with errors back to the form
		// Ensure the .Habit in edit.tmpl reflects the current values from habitToUpdate for repopulation
		errorTemplateData.Frequency = frequencyPathValue // The original frequency context of the edit page
		errorTemplateData.PermittedFrequencies = []string{"daily", "weekly"}
		errorTemplateData.IsAuthenticated = true

		// Repopulate FormData for the template
		formData := make(map[string]string)
		for key, values := range r.PostForm { // Use r.PostForm to get all submitted values
			if len(values) > 0 {
				formData[key] = values[0]
			}
		}
		errorTemplateData.FormData = formData
		// Override with values from habitToUpdate for consistency if needed
		errorTemplateData.FormData["title"] = habitToUpdate.Title
		errorTemplateData.FormData["description"] = habitToUpdate.Description
		errorTemplateData.FormData["goal"] = habitToUpdate.Goal
		// Frequency in FormData should be the one submitted
		errorTemplateData.FormData["frequency"] = habitToUpdate.Frequency

		err = app.render(w, r, http.StatusUnprocessableEntity, "edit.tmpl", errorTemplateData)
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	// Perform the update. Model.Update now takes habit *Habit which includes UserID
	// and the SQL query will have `WHERE id = $X AND user_id = $Y`
	err = app.habits.Update(habitToUpdate)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) { // If Model.Update returns this (e.g. 0 rows affected)
			app.notFound(w) // Habit might have been deleted by another request or didn't match user
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// Redirect to the page of the *new* frequency of the habit
	app.session.Put(r, "flash", "Habit updated successfully.")
	http.Redirect(w, r, "/"+habitToUpdate.Frequency, http.StatusSeeOther)
}

// deleteHabitHandler removes a habit if it belongs to the user
func (app *application) deleteHabitHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.logger.Error("Invalid habit ID for delete", "error", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Optional: Fetch habit first to log its details or ensure it exists before delete attempt
	// habit, err := app.habits.GetByID(id)
	// if err != nil { ... handle not found ... }
	// if habit.UserID != userID { ... handle forbidden ... return }

	app.logger.Info("Attempting to delete habit", "id", id, "userID", userID)

	// Model.Delete now takes id and userID
	err = app.habits.Delete(id, userID)
	if err != nil {
		app.logger.Error("Failed to delete habit", "id", id, "userID", userID, "error", err)
		if errors.Is(err, data.ErrRecordNotFound) {
			// This means the habit didn't exist or didn't belong to the user.
			// For a DELETE request, responding with 200 OK or 204 No Content is often fine
			// even if the resource was already gone, to make it idempotent.
			// Or, if HTMX expects a specific target to be removed, a 404 might break HTMX.
			// If it's an HTMX request, just returning OK might be best so HTMX can remove the element.
			if isHTMXRequest(r) {
				w.WriteHeader(http.StatusOK)
				return
			}
			app.notFound(w) // For non-HTMX, a 404 is clearer.
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.logger.Info("Habit deleted successfully", "id", id, "userID", userID)

	if isHTMXRequest(r) {
		// For HTMX, typically you'd have swapped out the element or a parent.
		// Returning 200 OK is usually sufficient as the hx-swap would have handled the UI.
		w.WriteHeader(http.StatusOK)
	} else {
		// For non-HTMX, redirect. Need to know where to redirect.
		// The original frequency might have been part of the URL or form data.
		// For simplicity, redirecting to user's home or a default habit page.
		// This part might need adjustment based on how delete is triggered in non-HTMX.
		// Assuming a path like /habits/delete/{frequency}/{id} would be better for non-HTMX.
		// Since your current route is /habits/delete/{id}, we don't have frequency easily.
		http.Redirect(w, r, "/"+r.PathValue("frequency"), http.StatusSeeOther)
	}
}

// progressHandler calculates and returns completion progress for the authenticated user
func (app *application) progressHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		// For HTMX, returning an empty progress or 0% might be okay
		// For non-HTMX, this endpoint might not be directly accessed.
		if isHTMXRequest(r) {
			w.Header().Set("Content-Type", "text/html")
			// Return 0 progress if not authenticated for an HTMX request
			w.Write([]byte(`<div class="bg-indigo-500 h-4 rounded-full" style="width: 0%"></div>`))
		} else {
			app.clientError(w, http.StatusUnauthorized)
		}
		return
	}

	frequency := r.PathValue("frequency")
	if frequency != "daily" && frequency != "weekly" {
		app.notFound(w)
		return
	}

	// Get habits for the specific user and frequency
	habits, err := app.habits.GetAllByFrequency(userID, frequency) // <<< PASS userID
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var completed, total int
	today := time.Now().Format("2006-01-02")

	for _, habit := range habits { // Iterate over user's habits
		entries, err := app.habits.GetEntries(habit.ID, time.Now(), time.Now())
		if err == nil && len(entries) > 0 {
			for _, entry := range entries {
				if entry.EntryDate.Format("2006-01-02") == today && entry.Status == "completed" {
					completed++
					break
				}
			}
		}
		total++
	}

	progress := 0
	if total > 0 {
		progress = (completed * 100) / total
	}

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="bg-indigo-500 h-4 rounded-full" style="width: ` + strconv.Itoa(progress) + `%"></div>`))
		return
	}

	// Regular response (if this handler is ever called non-HTMX for progress)
	data := NewTemplateData()
	data.Progress = progress
	// data.IsAuthenticated = true // If rendering a full page via this route
	app.render(w, r, http.StatusOK, "partials/progress_bar.tmpl", data) // Or a full page if needed
}

func (app *application) signupUserForm(w http.ResponseWriter, r *http.Request) {
	data := NewTemplateData()
	data.Title = "Sign Up"

	// signup.tmpl is now a standalone page
	err := app.render(w, r, http.StatusOK, "signup.tmpl", data)
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
		errRender := app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", formData)
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
			errRender := app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", formData)
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
	err := app.render(w, r, http.StatusOK, "login.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) loginUser(w http.ResponseWriter, r *http.Request) {
	app.logger.Info("Login attempt started") // <-- ADD THIS

	err := r.ParseForm()
	if err != nil {
		app.logger.Error("Login: Failed to parse form", "error", err) // <-- ADD THIS
		app.clientError(w, http.StatusBadRequest)
		return
	}

	email := r.PostForm.Get("email")
	passwordInput := r.PostForm.Get("password")

	// Log the received credentials (BE CAREFUL WITH PASSWORDS IN PRODUCTION LOGS)
	// For debugging, this is okay, but remove or hash passwords for production.
	app.logger.Info("Login attempt", "email", email, "password_provided_length", len(passwordInput)) // <-- MODIFIED LOGGING

	v := validator.NewValidator()

	if !validator.NotBlank(email) || !validator.NotBlank(passwordInput) {
		app.logger.Info("Login: Blank email or password detected by NotBlank check") // <-- ADD THIS
		v.AddError("generic", "Both email and password must be provided.")
	}

	if !v.ValidData() {
		app.logger.Info("Login: Validation failed (e.g., blank fields)", "errors", v.Errors) // <-- ADD THIS
		data := NewTemplateData()
		data.Title = "Login - Error"
		data.FormData = map[string]string{"email": email}
		data.FormErrors = v.Errors

		errRender := app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		if errRender != nil {
			app.logger.Error("Login: Failed to render login.tmpl on validation error", "render_error", errRender) // <-- ADD THIS
			app.serverError(w, r, errRender)
		}
		return
	}

	app.logger.Info("Login: Attempting to authenticate user", "email", email) // <-- ADD THIS
	id, err := app.users.Authenticate(email, passwordInput)
	if err != nil {
		if errors.Is(err, data.ErrInvalidCredentials) {
			app.logger.Info("Login: Authentication failed - Invalid Credentials", "email", email, "auth_error", err) // <-- MODIFIED LOGGING
			v.AddError("generic", "Invalid email or password.")

			data := NewTemplateData()
			data.Title = "Login - Error"
			data.FormData = map[string]string{"email": email}
			data.FormErrors = v.Errors

			errRender := app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
			if errRender != nil {
				app.logger.Error("Login: Failed to render login.tmpl on auth error", "render_error", errRender) // <-- ADD THIS
				app.serverError(w, r, errRender)
			}
		} else {
			// Log any other unexpected errors from Authenticate
			app.logger.Error("Login: Unexpected error during authentication", "email", email, "auth_error", err) // <-- MODIFIED LOGGING
			app.serverError(w, r, err)
		}
		return
	}

	app.logger.Info("Login: Authentication successful", "userID", id, "email", email) // <-- MODIFIED LOGGING
	app.session.Put(r, "authenticatedUserID", id)
	app.session.Put(r, "flash", "You have been logged in successfully!")
	http.Redirect(w, r, "/apphome", http.StatusSeeOther)
}

// logoutUserHandler handles user logout
func (app *application) logoutUserHandler(w http.ResponseWriter, r *http.Request) {
	// Remove the authenticatedUserID key from the session data.
	app.session.Remove(r, "authenticatedUserID")

	// Put a flash message in the session to inform the user.
	app.session.Put(r, "flash", "You have been logged out successfully.")

	// Redirect the user to the login page.
	// You could also redirect to the home page ("/") if preferred.
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
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
