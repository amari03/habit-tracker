package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings" // Make sure this is imported
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
	userID := app.authenticatedUserID(r)
	// User should be authenticated due to middleware, but good to have the ID

	templatePageData := NewTemplateData()
	templatePageData.Title = "Home"
	templatePageData.IsAuthenticated = true

	// Fetch user details to get the name
	if userID != 0 {
		user, err := app.users.Get(userID) // Assuming app.users.Get(id) exists and returns (*data.User, error)
		if err != nil {
			// Log the error, but don't necessarily block the page from rendering.
			// The welcome message will just be generic.
			app.logger.Error("Failed to get user details for home page", "userID", userID, "error", err)
		} else if user != nil {
			templatePageData.UserName = user.Name
		}
	}

	err := app.render(w, r, http.StatusOK, "home.tmpl", templatePageData)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// landingHandler renders the landing page
func (app *application) landingPageHandler(w http.ResponseWriter, r *http.Request) {
	data := NewTemplateData()
	data.Title = "Welcome"

	err := app.render(w, r, http.StatusOK, "landing.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// showHabitFormPageHandler renders the page with the habit creation form (daily.tmpl or weekly.tmpl)
func (app *application) showHabitFormPageHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		// This should be caught by requireAuthentication middleware, but good practice
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	var frequency string
	var templateName string

	// Determine frequency from URL path
	// Note: r.URL.Path will be like "/daily" or "/weekly"
	switch r.URL.Path {
	case "/daily":
		frequency = "daily"
		templateName = "daily.tmpl"
	case "/weekly":
		frequency = "weekly"
		templateName = "weekly.tmpl"
	default:
		app.notFound(w)
		return
	}

	templatePageData := NewTemplateData()
	templatePageData.Title = "Create " + frequency + " Habit"
	templatePageData.Frequency = frequency
	templatePageData.IsAuthenticated = true // User is authenticated
	templatePageData.Flash = app.session.PopString(r, "flash")
	// FormData and FormErrors might be needed if we redirect back to this form page with errors from elsewhere
	// For a fresh form, they'll be empty/nil which is fine.

	err := app.render(w, r, http.StatusOK, templateName, templatePageData)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// showHabitEntriesPageHandler renders the page displaying habit entries in a table
func (app *application) showHabitEntriesPageHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	var frequency string
	// Example path: /daily/entries or /weekly/entries
	// We need to extract "daily" or "weekly"
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || (pathParts[0] != "daily" && pathParts[0] != "weekly") || pathParts[1] != "entries" {
		app.notFound(w)
		return
	}
	frequency = pathParts[0]

	habits, err := app.habits.GetAllByFrequency(userID, frequency)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	habitPtrs := make([]*data.Habit, len(habits))
	today := time.Now().Format("2006-01-02")

	var completedForProgress, totalForProgress int
	for i := range habits {
		habitPtrs[i] = &habits[i]
		entries, entryErr := app.habits.GetEntries(habits[i].ID, time.Now(), time.Now()) // Get today's entries
		if entryErr == nil && len(entries) > 0 {
			// Assuming GetEntries orders by date desc, so first entry is the latest for the day
			if entries[0].EntryDate.Format("2006-01-02") == today {
				habitPtrs[i].TodayStatus = entries[0].Status
				if entries[0].Status == "completed" {
					completedForProgress++
				}
			}
		}
		totalForProgress++
	}

	currentProgress := 0
	if totalForProgress > 0 {
		currentProgress = (completedForProgress * 100) / totalForProgress
	}

	templatePageData := NewTemplateData()
	templatePageData.Title = frequency + " Habits Entries"
	templatePageData.Habits = habitPtrs
	templatePageData.Frequency = frequency
	templatePageData.IsAuthenticated = true
	templatePageData.Progress = currentProgress // Initial progress for the page
	templatePageData.Flash = app.session.PopString(r, "flash")

	err = app.render(w, r, http.StatusOK, "entries.tmpl", templatePageData)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// createHabitHandler handles new habit creation
func (app *application) createHabitHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	habit := &data.Habit{
		UserID:      userID,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Frequency:   r.FormValue("frequency"),
		Goal:        r.FormValue("goal"),
	}

	v := validator.NewValidator()
	data.ValidateHabit(v, habit)

	if !v.ValidData() {
		formTemplateData := NewTemplateData()
		formTemplateData.FormErrors = v.Errors
		formTemplateData.FormData = map[string]string{
			"title":       habit.Title,
			"description": habit.Description,
			"goal":        habit.Goal,
		}
		formTemplateData.Frequency = habit.Frequency
		formTemplateData.IsAuthenticated = (userID != 0)
		formTemplateData.Title = "Create " + habit.Frequency + " Habit - Error"

		// Determine which template to re-render (daily.tmpl or weekly.tmpl)
		var formTemplateName string
		if habit.Frequency == "daily" {
			formTemplateName = "daily.tmpl"
		} else if habit.Frequency == "weekly" {
			formTemplateName = "weekly.tmpl"
		} else {
			app.serverError(w, r, errors.New("invalid frequency on validation error"))
			return
		}
		err := app.render(w, r, http.StatusUnprocessableEntity, formTemplateName, formTemplateData)
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

	app.session.Put(r, "flash", "Habit created successfully!")

	// Redirect to the entries page for the habit's frequency
	redirectURL := "/" + habit.Frequency + "/entries"
	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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
		app.notFound(w)
		return
	}

	entry := &data.HabitEntry{
		HabitID:   habitID,
		EntryDate: time.Now(),
		Status:    r.FormValue("status"),
		Notes:     r.FormValue("notes"),
	}

	err = app.habits.LogEntry(entry)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	redirectURL := "/" + habit.Frequency + "/entries"
	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

// editHabitHandler shows the edit form if the habit belongs to the user
func (app *application) editHabitHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	frequencyPathValue := r.PathValue("frequency")
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

	if habit.UserID != userID {
		app.notFound(w)
		return
	}

	editTemplateData := NewTemplateData()
	editTemplateData.Title = "Edit Habit"
	editTemplateData.Habit = habit
	editTemplateData.Frequency = frequencyPathValue
	editTemplateData.PermittedFrequencies = []string{"daily", "weekly"}
	editTemplateData.IsAuthenticated = true
	editTemplateData.FormData = map[string]string{
		"title":       habit.Title,
		"description": habit.Description,
		"frequency":   habit.Frequency,
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

	frequencyPathValue := r.PathValue("frequency")
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

	habitToUpdate := &data.Habit{
		ID:          id,
		UserID:      userID,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Frequency:   r.FormValue("frequency"),
		Goal:        r.FormValue("goal"),
	}

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
		app.notFound(w)
		return
	}

	v := validator.NewValidator()
	data.ValidateHabit(v, habitToUpdate)
	if !v.ValidData() {
		errorTemplateData := NewTemplateData()
		errorTemplateData.Title = "Edit Habit - Error"
		errorTemplateData.FormErrors = v.Errors
		errorTemplateData.Habit = habitToUpdate
		errorTemplateData.Frequency = frequencyPathValue
		errorTemplateData.PermittedFrequencies = []string{"daily", "weekly"}
		errorTemplateData.IsAuthenticated = true
		errorTemplateData.FormData = map[string]string{
			"title":       habitToUpdate.Title,
			"description": habitToUpdate.Description,
			"frequency":   habitToUpdate.Frequency,
			"goal":        habitToUpdate.Goal,
		}
		err = app.render(w, r, http.StatusUnprocessableEntity, "edit.tmpl", errorTemplateData)
		if err != nil {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.habits.Update(habitToUpdate)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.session.Put(r, "flash", "Habit updated successfully.")
	http.Redirect(w, r, "/"+habitToUpdate.Frequency+"/entries", http.StatusSeeOther)
}

// deleteHabitHandler removes a habit if it belongs to the user
func (app *application) deleteHabitHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	frequency := r.PathValue("frequency")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	if frequency != "daily" && frequency != "weekly" {
		app.notFound(w)
		return
	}

	err = app.habits.Delete(id, userID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			if isHTMXRequest(r) {
				w.WriteHeader(http.StatusOK)
				return
			}
			app.notFound(w)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.session.Put(r, "flash", "Habit deleted successfully.")
	redirectURL := "/" + frequency + "/entries"

	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

// progressHandler calculates and returns completion progress for the authenticated user
func (app *application) progressHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.authenticatedUserID(r)
	if userID == 0 {
		if isHTMXRequest(r) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<div class="bg-indigo-500 h-4 rounded-full" style="width: 0%">0%</div>`))
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

	habits, err := app.habits.GetAllByFrequency(userID, frequency)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	var completed, total int
	today := time.Now().Format("2006-01-02")

	for _, habit := range habits {
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

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="progress-bar" style="width: ` + strconv.Itoa(progress) + `%;">` + strconv.Itoa(progress) + `%</div>`))
}

func (app *application) signupUserForm(w http.ResponseWriter, r *http.Request) {
	data := NewTemplateData()
	data.Title = "Sign Up"
	err := app.render(w, r, http.StatusOK, "signup.tmpl", data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) signupUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	name := r.PostForm.Get("name")
	email := r.PostForm.Get("email")
	passwordInput := r.PostForm.Get("password")

	v := validator.NewValidator()
	user := &data.User{
		Name:  name,
		Email: email,
	}
	data.ValidateUser(v, user)
	v.Check(validator.NotBlank(passwordInput), "password", "Password must be provided")
	v.Check(validator.MinLength(passwordInput, 8), "password", "Password must be at least 8 characters long")
	v.Check(validator.MaxLength(passwordInput, 72), "password", "Password must not be more than 72 characters")

	if !v.ValidData() {
		formData := NewTemplateData()
		formData.Title = "Sign Up - Error"
		formData.FormErrors = v.Errors
		formData.FormData = map[string]string{
			"name":  name,
			"email": email,
		}
		errRender := app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", formData)
		if errRender != nil {
			app.serverError(w, r, errRender)
		}
		return
	}

	err = user.Password.Set(passwordInput)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	user.Active = true

	err = app.users.Insert(user)
	if err != nil {
		if errors.Is(err, data.ErrDuplicateEmail) {
			v.AddError("email", "Email address is already registered")
			formData := NewTemplateData()
			formData.Title = "Sign Up - Error"
			formData.FormErrors = v.Errors
			formData.FormData = map[string]string{
				"name":  name,
				"email": email,
			}
			errRender := app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", formData)
			if errRender != nil {
				app.serverError(w, r, errRender)
			}
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.session.Put(r, "flash", "Your signup was successful! Please log in.")
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) loginUserForm(w http.ResponseWriter, r *http.Request) {
	data := NewTemplateData()
	data.Title = "Login"
	data.Flash = app.session.PopString(r, "flash")
	err := app.render(w, r, http.StatusOK, "login.tmpl", data)
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
	v := validator.NewValidator()

	if !validator.NotBlank(email) || !validator.NotBlank(passwordInput) {
		v.AddError("generic", "Both email and password must be provided.")
	}

	if !v.ValidData() {
		data := NewTemplateData()
		data.Title = "Login - Error"
		data.FormData = map[string]string{"email": email}
		data.FormErrors = v.Errors
		errRender := app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		if errRender != nil {
			app.serverError(w, r, errRender)
		}
		return
	}

	id, err := app.users.Authenticate(email, passwordInput)
	if err != nil {
		if errors.Is(err, data.ErrInvalidCredentials) {
			v.AddError("generic", "Invalid email or password.")
			data := NewTemplateData()
			data.Title = "Login - Error"
			data.FormData = map[string]string{"email": email}
			data.FormErrors = v.Errors
			errRender := app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
			if errRender != nil {
				app.serverError(w, r, errRender)
			}
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	app.session.Put(r, "authenticatedUserID", id)
	app.session.Put(r, "flash", "You have been logged in successfully!")
	http.Redirect(w, r, "/apphome", http.StatusSeeOther)
}

func (app *application) logoutUserHandler(w http.ResponseWriter, r *http.Request) {
	app.session.Remove(r, "authenticatedUserID")
	app.session.Put(r, "flash", "You have been logged out successfully.")
	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
	if isHTMXRequest(r) && w.Header().Get("Content-Type") == "" {
		w.Header().Set("HX-Retarget", "body")
		w.Header().Set("HX-Reswap", "innerHTML")
	}
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}
