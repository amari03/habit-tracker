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
)

type Habit struct {
	ID          int64     `json:"id"`
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

	v.Check(validator.MaxLength(h.Description, 1000), "description", "must not be more than 1000 characters")
	v.Check(validator.MaxLength(h.Goal, 100), "goal", "must not be more than 100 characters")
}

type HabitModel struct {
	DB *sql.DB
}

// Insert a new habit
func (m *HabitModel) Insert(habit *Habit) error {
	query := `
		INSERT INTO habits (title, description, frequency, goal)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query,
		habit.Title,
		habit.Description,
		habit.Frequency,
		habit.Goal,
	).Scan(&habit.ID, &habit.CreatedAt, &habit.UpdatedAt)
}

// GetAllByFrequency returns all habits with matching frequency ("daily" or "weekly")
func (m *HabitModel) GetAllByFrequency(frequency string) ([]Habit, error) {
	query := `
		SELECT id, title, description, frequency, goal, created_at, updated_at
		FROM habits
		WHERE frequency = $1
		ORDER BY created_at DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, frequency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []Habit

	for rows.Next() {
		var h Habit
		err := rows.Scan(&h.ID, &h.Title, &h.Description, &h.Frequency, &h.Goal, &h.CreatedAt, &h.UpdatedAt)
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
        SELECT id, title, description, frequency, goal, created_at, updated_at
        FROM habits
        WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var habit Habit
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&habit.ID,
		&habit.Title,
		&habit.Description,
		&habit.Frequency,
		&habit.Goal,
		&habit.CreatedAt,
		&habit.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &habit, nil
}

// Update a habit
func (m *HabitModel) Update(habit *Habit) error {
	query := `
		UPDATE habits
		SET title = $1, description = $2, frequency = $3, goal = $4, updated_at = NOW()
		WHERE id = $5`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query,
		habit.Title,
		habit.Description,
		habit.Frequency,
		habit.Goal,
		habit.ID,
	)
	return err
}

// Delete a habit
func (m *HabitModel) Delete(id int64) error {
	query := `DELETE FROM habits WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id)
	return err
}

// method for toggling completion status
func (m *HabitModel) ToggleCompletion(id int64) error {
	query := `
		UPDATE habits
		SET completed = NOT completed,
			updated_at = NOW()
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id)
	return err
}

// Add these methods to HabitModel in habit.go

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
