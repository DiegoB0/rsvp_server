package guests

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/diegob0/rspv_backend/internal/utils"
	"github.com/lib/pq"
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
		&guest.TicketGenerated,
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
	rows, err := s.db.Query("SELECT id, full_name, additionals, confirm_attendance, table_id, created_at, ticket_generated   FROM guests WHERE name=$1", name)
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
	rows, err := s.db.Query("SELECT id, full_name, additionals, confirm_attendance, table_id, created_at, ticket_generated FROM guests WHERE id=$1", id)
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

func (s *Store) GetGuests(params types.PaginationParams) (*types.PaginatedResult[*types.Guest], error) {
	var whereClause string
	var args []interface{}
	orderBy := "id"

	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		whereClause = " WHERE full_name ILIKE $1"
		args = append(args, "%"+strings.TrimSpace(*params.Search)+"%")
	}

	baseQuery := `
		SELECT id, full_name, additionals, confirm_attendance, table_id, created_at, ticket_generated
		FROM guests
	` + whereClause

	countQuery := `SELECT COUNT(*) FROM guests` + whereClause

	return utils.Paginate(s.db, baseQuery, countQuery, scanRowIntoGuests, params, orderBy, args...)
}

func (s *Store) GetUnassignedGuests(params types.PaginationParams) (*types.PaginatedResult[*types.Guest], error) {
	var andWhere string
	var args []interface{}
	orderBy := "id"
	whereClause := " WHERE table_id IS NULL"

	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		andWhere = " AND full_name ILIKE $1"
		args = append(args, "%"+strings.TrimSpace(*params.Search)+"%")
	}

	baseQuery := `
		SELECT id, full_name, additionals, confirm_attendance, table_id, created_at, ticket_generated
		FROM guests
	` + whereClause + andWhere

	countQuery := `SELECT COUNT(*) FROM guests` + whereClause + andWhere

	return utils.Paginate(s.db, baseQuery, countQuery, scanRowIntoGuests, params, orderBy, args...)
}

func (s *Store) CreateGuest(guest types.Guest) error {
	normalized := strings.ToLower(guest.FullName)

	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM guests WHERE LOWER(full_name) = $1)", normalized).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for existing guest: %w", err)
	}
	if exists {
		return fmt.Errorf("guest with name '%s' already exists", guest.FullName)
	}

	_, err = s.db.Exec("INSERT INTO guests (full_name, additionals, confirm_attendance) VALUES ($1, $2, $3)", guest.FullName, guest.Additionals, guest.ConfirmAttendance)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteGuest(id int) error {
	var tableID sql.NullInt64

	err := s.db.QueryRow(`
	SELECT table_id FROM guests WHERE id = $1
		`, id).Scan(&tableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("guest with id %d not found", id)
		}
		return err

	}

	// Unassign if any
	if tableID.Valid {
		if err := s.UnassignGuest(id); err != nil {
			return err
		}
	}

	// Delete guest
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
	var tableID *int
	err := s.db.QueryRow(`
		SELECT table_id FROM guests WHERE id = $1
		`, guest.ID).Scan(&tableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("guest with id %d was not found", guest.ID)
		}
	}

	if tableID != nil {
		return fmt.Errorf("cannot update guest: %v assigned to a table (id=%d). Unassign the guest from the table first", guest.ID, *tableID)
	}

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

	if oldTableID.Valid && int(oldTableID.Int32) == tableID {
		return fmt.Errorf("guest %d is already assigned to table %d", guestID, tableID)
	}

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

// Methods to get the tickets per guest
func (s *Store) GetTicketsPerGuest(guestID int) ([]types.GuestWithTickets, error) {
	rows, err := s.db.Query("SELECT id, full_name, additionals, confirm_attendance, table_id, created_at, ticket_generated, qr_code_urls FROM guests WHERE id = $1", guestID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var guests []types.GuestWithTickets

	for rows.Next() {
		guest, err := scanRowIntoGuestWithTickets(rows)
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

// Helper
func scanRowIntoGuestWithTickets(rows *sql.Rows) (*types.GuestWithTickets, error) {
	guest := new(types.GuestWithTickets)
	var tableId sql.NullInt64

	err := rows.Scan(
		&guest.ID,
		&guest.FullName,
		&guest.Additionals,
		&guest.ConfirmAttendance,
		&tableId,
		&guest.CreatedAt,
		&guest.TicketGenerated,
		pq.Array(&guest.QrCodeUrls),
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
