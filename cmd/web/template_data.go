package main

import (
	"github.com/amari03/habit-tracker/internal/data"
	"time"
)

type TemplateData struct {
	Title                string
	Year                 int
	HeaderText           string
	PermittedFrequencies []string
	FormErrors           map[string]string
	FormData             map[string]string
	Habits               []*data.Habit // Changed from DailyHabits/WeeklyHabits to generic Habits
	Habit                *data.Habit   // Single habit (for edit/view)
	Progress             int           // For progress bar
	Frequency            string        // "daily" or "weekly"
	Flash                string        // For flash messages
	IsAuthenticated      bool          // For authentication check
	CSRFToken            string        // CSRF token for forms
	UserName             string        // <<< ADD THIS FIELD for the user's name
}

func NewTemplateData() *TemplateData {
	return &TemplateData{
		FormErrors:      make(map[string]string),
		FormData:        make(map[string]string),
		Year:            time.Now().Year(), // Added default year
		IsAuthenticated: false,             // Default to false
		CSRFToken:       "",                // Default to empty string
		UserName:        "",                // Initialize UserName
	}
}
