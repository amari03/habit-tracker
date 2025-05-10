package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/amari03/habit-tracker/internal/validator"
)

// Add this error definition
var (
	ErrRecordNotFound = errors.New("record not found")
	// You might want to add an error for unauthorized access later
	// ErrForbiddenAccess = errors.New("forbidden access")
)

type Habit struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Frequency   string    `json:"frequency"` // daily, weekly
	Goal        string    `json:"goal"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TodayStatus string    `json:"today_status,omitempty"`
}

func ValidateHabit(v *validator.Validator, h *Habit) {
	v.Check(validator.NotBlank(h.Title), "title", "must be provided")
	v.Check(validator.MaxLength(h.Title, 255), "title", "must not be more than 255 characters")

	v.Check(validator.NotBlank(h.Frequency), "frequency", "must be provided")
	v.Check(validator.PermittedValue(h.Frequency, "daily", "weekly"), "frequency", "must be 'daily' or 'weekly'")

	v.Check(validator.NotBlank(h.Description), "description", "must be provided")
	v.Check(validator.MaxLength(h.Description, 1000), "description", "must not be more than 1000 characters")

	v.Check(validator.NotBlank(h.Goal), "goal", "must be provided")
	v.Check(validator.MaxLength(h.Goal, 100), "goal", "must not be more than 100 characters")
}

type HabitModel struct {
	DB *sql.DB
}

// Insert a new habit
func (m *HabitModel) Insert(habit *Habit) error {
	query := `
		INSERT INTO habits (user_id, title, description, frequency, goal) -- Add user_id
        VALUES ($1, $2, $3, $4, $5)                                  -- Add $1 for user_id
        RETURNING id, created_at, updated_at`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query,
		habit.UserID, // Pass UserID
		habit.Title,
		habit.Description,
		habit.Frequency,
		habit.Goal,
	).Scan(&habit.ID, &habit.CreatedAt, &habit.UpdatedAt)
}

// GetAllByFrequency returns all habits for a given user with matching frequency
func (m *HabitModel) GetAllByFrequency(userID int64, frequency string) ([]Habit, error) {
	query := `
		SELECT id, user_id, title, description, frequency, goal, created_at, updated_at -- Add user_id
		FROM habits
		WHERE user_id = $1 AND frequency = $2 -- Filter by user_id and frequency
		ORDER BY created_at DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID, frequency) // Pass userID
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []Habit

	for rows.Next() {
		var h Habit
		err := rows.Scan(
			&h.ID,
			&h.UserID, // Scan UserID
			&h.Title,
			&h.Description,
			&h.Frequency,
			&h.Goal,
			&h.CreatedAt,
			&h.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return habits, nil
}

// GetByID returns a single habit by its ID
func (m *HabitModel) GetByID(id int64) (*Habit, error) {
	query := `
        SELECT id, user_id, title, description, frequency, goal, created_at, updated_at -- Add user_id
        FROM habits
        WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var habit Habit
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&habit.ID,
		&habit.UserID, // Scan UserID
		&habit.Title,
		&habit.Description,
		&habit.Frequency,
		&habit.Goal,
		&habit.CreatedAt,
		&habit.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &habit, nil
}

// Update a habit for a specific user
func (m *HabitModel) Update(habit *Habit) error {
	query := `
		UPDATE habits
		SET title = $1, description = $2, frequency = $3, goal = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6` // Add user_id to WHERE clause for ownership

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query,
		habit.Title,
		habit.Description,
		habit.Frequency,
		habit.Goal,
		habit.ID,
		habit.UserID, // Pass UserID for the WHERE clause
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// This could mean the habit ID didn't exist OR it didn't belong to the user.
		// The handlers will need to distinguish this, perhaps by first calling GetByID.
		return ErrRecordNotFound // Or a more specific "not found or not authorized"
	}

	return nil
}

// Delete a habit for a specific user
func (m *HabitModel) Delete(id int64, userID int64) error {
	query := `DELETE FROM habits WHERE id = $1 AND user_id = $2` // Add user_id to WHERE

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id, userID) // Pass userID
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// This means the habit ID didn't exist OR it didn't belong to the user.
		return ErrRecordNotFound // Or a more specific "not found or not authorized"
	}
	return nil
}

// method for toggling completion status
/*func (m *HabitModel) ToggleCompletion(id int64) error {
	query := `
		UPDATE habits
		SET completed = NOT completed,
			updated_at = NOW()
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id)
	return err
}*/

// GetEntries returns all entries for a habit within a date range
func (m *HabitModel) GetEntries(habitID int64, from, to time.Time) ([]HabitEntry, error) {
	query := `
        SELECT id, habit_id, entry_date, status, notes, created_at
        FROM habit_entries
        WHERE habit_id = $1 AND entry_date BETWEEN $2 AND $3
        ORDER BY entry_date DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, habitID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HabitEntry
	for rows.Next() {
		var e HabitEntry
		err := rows.Scan(
			&e.ID,
			&e.HabitID,
			&e.EntryDate,
			&e.Status,
			&e.Notes,
			&e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// LogEntry creates a new habit entry
func (m *HabitModel) LogEntry(entry *HabitEntry) error {
	query := `
        INSERT INTO habit_entries (habit_id, entry_date, status, notes)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query,
		entry.HabitID,
		entry.EntryDate,
		entry.Status,
		entry.Notes,
	).Scan(&entry.ID, &entry.CreatedAt)
}
