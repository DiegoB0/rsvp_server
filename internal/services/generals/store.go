package generals

import (
	"database/sql"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) AssignGeneral(generalID int, tableID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldTableID sql.NullInt32

	err = tx.QueryRow(`
		SELECT table_id 
		FROM generals 
		WHERE id = $1
	`, generalID).Scan(&oldTableID)
	if err != nil {
		return fmt.Errorf("general ticket not found: %w", err)
	}

	const totalSeats = 1

	var capacity int
	err = tx.QueryRow(`
		SELECT capacity 
		FROM tables 
		WHERE id = $1
	`, tableID).Scan(&capacity)
	if err != nil {
		return fmt.Errorf("table not found: %w", err)
	}

	if capacity < totalSeats {
		return fmt.Errorf("not enough space at table %d: needed %d, available %d", tableID, totalSeats, capacity)
	}

	if oldTableID.Valid && int(oldTableID.Int32) != tableID {

		_, err := tx.Exec(`
			UPDATE tables 
			SET capacity = capacity + $1 
			WHERE id = $2
		`, totalSeats, oldTableID.Int32)
		if err != nil {
			return fmt.Errorf("failed to restore old table capacity: %w", err)
		}
	}

	_, err = tx.Exec(`
		UPDATE tables 
		SET capacity = capacity - $1 
		WHERE id = $2
	`, totalSeats, tableID)
	if err != nil {
		return fmt.Errorf("failed to update new table capacity: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE generals 
		SET table_id = $1 
		WHERE id = $2
	`, tableID, generalID)
	if err != nil {
		return fmt.Errorf("failed to assign general to table: %w", err)
	}

	return tx.Commit()
}

func (s *Store) UnassignGeneral(generalID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tableID sql.NullInt32

	err = tx.QueryRow(`
		SELECT table_id
		FROM generals
		WHERE id = $1
	`, generalID).Scan(&tableID)
	if err != nil {
		return fmt.Errorf("general not found: %w", err)
	}

	if !tableID.Valid {
		return fmt.Errorf("general is not assigned to any table")
	}

	const totalSeats = 1

	_, err = tx.Exec(`
		UPDATE tables 
		SET capacity = capacity + $1 
		WHERE id = $2
	`, totalSeats, tableID.Int32)
	if err != nil {
		return fmt.Errorf("failed to update table capacity: %w", err)
	}

	// Unassign general
	_, err = tx.Exec(`
		UPDATE generals 
		SET table_id = NULL 
		WHERE id = $1
	`, generalID)
	if err != nil {
		return fmt.Errorf("failed to unassign general: %w", err)
	}

	return tx.Commit()
}

func (s *Store) DeleteLastGenerals(count int) error {
	if count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Fetch the last `count` generals (id + table_id)
	rows, err := tx.Query(`
		SELECT id, table_id 
		FROM generals 
		ORDER BY id DESC 
		LIMIT $1
	`, count)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		var tableID sql.NullInt64

		if err := rows.Scan(&id, &tableID); err != nil {
			return err
		}

		if tableID.Valid {
			return fmt.Errorf("cannot delete general %d: assigned to table %d", id, tableID.Int64)
		}

		ids = append(ids, id)
	}

	// 2. Check if we got enough generals
	if len(ids) < count {
		return fmt.Errorf("only %d generals exist, cannot delete %d", len(ids), count)
	}

	// 3. Delete the generals
	for _, id := range ids {
		_, err := tx.Exec(`DELETE FROM generals WHERE id = $1`, id)
		if err != nil {
			return fmt.Errorf("failed to delete general %d: %w", id, err)
		}
	}

	return tx.Commit()
}
