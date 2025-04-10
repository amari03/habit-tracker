package data

import (
	"context"
	"database/sql"
	"time"
	"errors"
)

type HabitEntry struct {
	ID        int64     `json:"id"`
	HabitID   int64     `json:"habit_id"`
	EntryDate time.Time `json:"entry_date"`
	Status    string    `json:"status"` // e.g., "completed", "skipped", "missed"
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type HabitEntryModel struct {
	DB *sql.DB
}

// Insert a new habit entry
func (m *HabitEntryModel) Insert(entry *HabitEntry) error {
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

func (m *HabitEntryModel) GetTodayStatus(habitID int64) (string, error) {
    query := `
        SELECT status 
        FROM habit_entries 
        WHERE habit_id = $1 AND entry_date = CURRENT_DATE
        LIMIT 1`

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    var status string
    err := m.DB.QueryRowContext(ctx, query, habitID).Scan(&status)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return "", nil
        }
        return "", err
    }
    return status, nil
}

// Get all entries for a specific habit
func (m *HabitEntryModel) GetByHabitID(habitID int64) ([]HabitEntry, error) {
	query := `
		SELECT id, habit_id, entry_date, status, notes, created_at
		FROM habit_entries
		WHERE habit_id = $1
		ORDER BY entry_date DESC`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, habitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HabitEntry
	for rows.Next() {
		var entry HabitEntry
		err := rows.Scan(
			&entry.ID,
			&entry.HabitID,
			&entry.EntryDate,
			&entry.Status,
			&entry.Notes,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// Update a habit entry
func (m *HabitEntryModel) Update(entry *HabitEntry) error {
    query := `
        UPDATE habit_entries
        SET status = $1, notes = $2
        WHERE id = $3
        RETURNING entry_date`

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    return m.DB.QueryRowContext(ctx, query, 
        entry.Status, 
        entry.Notes, 
        entry.ID,
    ).Scan(&entry.EntryDate)
}
// Delete a habit entry
func (m *HabitEntryModel) Delete(id int64) error {
	query := `DELETE FROM habit_entries WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, id)
	return err
}

// Completion Rate (for progress bar)
func (m *HabitEntryModel) GetCompletionRate(habitID int64) (float64, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'completed')::float / 
			NULLIF(COUNT(*), 0)::float
		FROM habit_entries
		WHERE habit_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var rate sql.NullFloat64
	err := m.DB.QueryRowContext(ctx, query, habitID).Scan(&rate)
	if err != nil {
		return 0, err
	}
	if rate.Valid {
		return rate.Float64, nil
	}
	return 0, nil
}

// Add this new method for bulk operations
func (m *HabitEntryModel) GetRecentCompletions(habitIDs []int64) (map[int64]bool, error) {
    query := `
        SELECT DISTINCT habit_id
        FROM habit_entries
        WHERE habit_id = ANY($1) 
        AND status = 'completed'
        AND entry_date = CURRENT_DATE`

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    rows, err := m.DB.QueryContext(ctx, query, habitIDs)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[int64]bool)
    for rows.Next() {
        var id int64
        if err := rows.Scan(&id); err != nil {
            return nil, err
        }
        result[id] = true
    }

    return result, nil
}