package generals

import (
	"database/sql"
	"errors"
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

func (s *Store) DeleteGeneral(id int) error {
	var tableID sql.NullInt64

	err := s.db.QueryRow(`
	SELECT table_id FROM generals WHERE id = $1
		`, id).Scan(&tableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("general with id %d not found", id)
		}
		return err

	}

	// Unassign if any
	if tableID.Valid {
		if err := s.UnassignGeneral(id); err != nil {
			return err
		}
	}

	// Delete general
	res, err := s.db.Exec("DELETE FROM generals WHERE id = $1", id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("generals with id %d not found", id)
	}
	return nil
}
