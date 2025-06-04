package guests

import (
	"database/sql"
	"fmt"

	"github.com/diegob0/rspv_backend/internal/types"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Helper function to scan each row of the table guests
func scanRowIntoGuests(rows *sql.Rows) (*types.Guest, error) {
	guest := new(types.Guest)
	var tableId sql.NullInt64

	err := rows.Scan(
		&guest.ID,
		&guest.FullName,
		&guest.Additionals,
		&guest.ConfirmAttendance,
		&tableId,
		&guest.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Assign tableId to guest.TableId (as *int)
	if tableId.Valid {
		val := int(tableId.Int64)
		guest.TableId = &val

	} else {
		guest.TableId = nil
	}

	return guest, nil
}

func (s *Store) GetGuestByName(name string) (*types.Guest, error) {
	rows, err := s.db.Query("SELECT * FROM guests WHERE name=$1", name)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	g := new(types.Guest)
	for rows.Next() {
		g, err = scanRowIntoGuests(rows)
		if err != nil {
			return nil, err
		}
	}

	if g.ID == 0 {
		return nil, fmt.Errorf("table not found")
	}

	return g, nil
}

func (s *Store) GetGuestByID(id int) (*types.Guest, error) {
	rows, err := s.db.Query("SELECT * FROM guests WHERE id=$1", id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	g := new(types.Guest)
	for rows.Next() {
		g, err = scanRowIntoGuests(rows)
		if err != nil {
			return nil, err
		}
	}

	if g.ID == 0 {
		return nil, fmt.Errorf("mesa not found")
	}

	return g, nil
}

func (s *Store) GetGuests() ([]types.Guest, error) {
	rows, err := s.db.Query("SELECT * FROM guests")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var guests []types.Guest

	for rows.Next() {
		guest, err := scanRowIntoGuests(rows)
		if err != nil {
			return nil, err
		}
		guests = append(guests, *guest)
	}

	// Handle if not users
	if len(guests) == 0 {
		return nil, fmt.Errorf("no guests found")
	}

	return guests, nil
}

func (s *Store) CreateGuest(guest types.Guest) error {
	_, err := s.db.Exec("INSERT INTO guests (full_name, additionals, confirm_attendance) VALUES ($1, $2, $3)", guest.FullName, guest.Additionals, guest.ConfirmAttendance)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteGuest(id int) error {
	res, err := s.db.Exec("DELETE FROM guests WHERE id = $1", id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("guest with id %d not found", id)
	}
	return nil
}

func (s *Store) UpdateGuest(guest *types.Guest) error {
	res, err := s.db.Exec(`
		UPDATE guests 
		SET full_name = $1, additionals = $2, confirm_attendance = $3
		WHERE id = $4
	`, guest.FullName, guest.Additionals, guest.ConfirmAttendance, guest.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("guest with id %d not found", guest.ID)
	}

	return nil
}

// Methods to assign and unassign guests to tables
func (s *Store) AssignGuest(guestID int, tableID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var oldTableID sql.NullInt32
	var additionals int

	// Fetch old table ID and additionals
	err = tx.QueryRow(`
		SELECT table_id, additionals 
		FROM guests 
		WHERE id = $1
	`, guestID).Scan(&oldTableID, &additionals)
	if err != nil {
		return fmt.Errorf("guest not found: %w", err)
	}

	totalSeats := 1 + additionals

	// Check if new table has enough capacity
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

	// Restore capacity to old table (if changing tables)
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

	// Subtract from new table
	_, err = tx.Exec(`
		UPDATE tables 
		SET capacity = capacity - $1 
		WHERE id = $2
	`, totalSeats, tableID)
	if err != nil {
		return fmt.Errorf("failed to update new table capacity: %w", err)
	}

	// Assign guest
	_, err = tx.Exec(`
		UPDATE guests 
		SET table_id = $1 
		WHERE id = $2
	`, tableID, guestID)
	if err != nil {
		return fmt.Errorf("failed to assign guest to table: %w", err)
	}

	return tx.Commit()
}

func (s *Store) UnassignGuest(guestID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tableID sql.NullInt32
	var additionals int

	// Get guest's current table and additionals
	err = tx.QueryRow(`
		SELECT table_id, additionals
		FROM guests
		WHERE id = $1
	`, guestID).Scan(&tableID, &additionals)
	if err != nil {
		return fmt.Errorf("guest not found: %w", err)
	}

	if !tableID.Valid {
		return fmt.Errorf("guest is not assigned to any table")
	}

	totalSeats := 1 + additionals

	// Increment table capacity
	_, err = tx.Exec(`
		UPDATE tables 
		SET capacity = capacity + $1 
		WHERE id = $2
	`, totalSeats, tableID.Int32)
	if err != nil {
		return fmt.Errorf("failed to update table capacity: %w", err)
	}

	// Unassign guest
	_, err = tx.Exec(`
		UPDATE guests 
		SET table_id = NULL 
		WHERE id = $1
	`, guestID)
	if err != nil {
		return fmt.Errorf("failed to unassign guest: %w", err)
	}

	return tx.Commit()
}

// Methods to get the tickets per user and the guest join to mesas
