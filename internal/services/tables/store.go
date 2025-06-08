package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/diegob0/rspv_backend/internal/types"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Helper function to scan each row of the table mesas
func scanRowIntoTable(rows *sql.Rows) (*types.Table, error) {
	table := new(types.Table)

	err := rows.Scan(
		&table.ID,
		&table.Name,
		&table.Capacity,
		&table.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return table, nil
}

func (s *Store) GetTableByID(id int) (*types.Table, error) {
	rows, err := s.db.Query("SELECT * FROM tables WHERE id=$1", id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	t := new(types.Table)
	for rows.Next() {
		t, err = scanRowIntoTable(rows)
		if err != nil {
			return nil, err
		}
	}

	if t.ID == 0 {
		return nil, fmt.Errorf("mesa not found")
	}

	return t, nil
}

func (s *Store) GetTables() ([]types.Table, error) {
	rows, err := s.db.Query("SELECT * FROM tables")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var tables []types.Table

	for rows.Next() {
		table, err := scanRowIntoTable(rows)
		if err != nil {
			return nil, err
		}
		tables = append(tables, *table)
	}

	// Handle if not users
	if len(tables) == 0 {
		return nil, fmt.Errorf("no mesas found")
	}

	return tables, nil
}

func (s *Store) GetTableByName(name string) (*types.Table, error) {
	rows, err := s.db.Query("SELECT * FROM tables WHERE name=$1", name)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	t := new(types.Table)
	for rows.Next() {
		t, err = scanRowIntoTable(rows)
		if err != nil {
			return nil, err
		}
	}

	if t.ID == 0 {
		return nil, fmt.Errorf("table not found")
	}

	return t, nil
}

func (s *Store) CreateTable(table types.Table) error {
	_, err := s.db.Exec("INSERT INTO tables (name, capacity) VALUES ($1, $2)", table.Name, table.Capacity)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteTable(id int) error {
	res, err := s.db.Exec("DELETE FROM tables WHERE id = $1", id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("table with id %d not found", id)
	}
	return nil
}

func (s *Store) UpdateTable(table *types.Table) error {
	res, err := s.db.Exec(`
		UPDATE tables 
		SET name = $1, capacity = $2
		WHERE id = $3
	`, table.Name, table.Capacity, table.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("table with id %d not found", table.ID)
	}

	return nil
}

// Stores for operations with join tables
func (s *Store) GetTablesWithGuests() ([]types.TableAndGuests, error) {
	query := `
		SELECT
			t.id, t.name, t.capacity, t.created_at,

			g.id, g.full_name, g.additionals, g.confirm_attendance, g.table_id, g.created_at
		FROM tables t

		LEFT JOIN guests g ON g.table_id = t.id
		ORDER BY t.id, g.id;
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tablesMap := make(map[int]*types.TableAndGuests)

	for rows.Next() {
		var (
			tID        int
			tName      string
			tCapacity  int
			tCreatedAt time.Time

			gID                sql.NullInt64
			gFullName          sql.NullString
			gAdditionals       sql.NullInt64
			gConfirmAttendance sql.NullBool
			gTableID           sql.NullInt64
			gCreatedAt         sql.NullTime
		)

		err := rows.Scan(
			&tID, &tName, &tCapacity, &tCreatedAt,
			&gID, &gFullName, &gAdditionals, &gConfirmAttendance, &gTableID, &gCreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Get or create the table
		table, exists := tablesMap[tID]
		if !exists {
			table = &types.TableAndGuests{
				ID:        tID,
				Name:      tName,
				Capacity:  tCapacity,
				CreatedAt: tCreatedAt,
				Guests:    []types.Guest{},
			}
			tablesMap[tID] = table
		}

		// If guest is present, add them
		if gID.Valid {
			guest := types.Guest{
				ID:                int(gID.Int64),
				FullName:          gFullName.String,
				Additionals:       int(gAdditionals.Int64),
				ConfirmAttendance: gConfirmAttendance.Bool,
				CreatedAt:         gCreatedAt.Time,
			}

			if gTableID.Valid {
				id := int(gTableID.Int64)
				guest.TableId = &id

			}

			table.Guests = append(table.Guests, guest)
		}
	}

	var result []types.TableAndGuests
	for _, t := range tablesMap {
		result = append(result, *t)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no tables with guests found")
	}

	return result, nil
}

func (s *Store) GetTableWithGuestsByID(tableID int) (*types.TableAndGuests, error) {
	query := `
		SELECT
			t.id, t.name, t.capacity, t.created_at,
			g.id, g.full_name, g.additionals, g.confirm_attendance, g.table_id, g.created_at
		FROM tables t
		LEFT JOIN guests g ON g.table_id = t.id
		WHERE t.id = $1
		ORDER BY g.id;

	`

	rows, err := s.db.Query(query, tableID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var result *types.TableAndGuests

	for rows.Next() {

		var (
			tID        int
			tName      string
			tCapacity  int
			tCreatedAt time.Time

			gID sql.NullInt64

			gFullName          sql.NullString
			gAdditionals       sql.NullInt64
			gConfirmAttendance sql.NullBool
			gTableID           sql.NullInt64

			gCreatedAt sql.NullTime
		)

		err := rows.Scan(
			&tID, &tName, &tCapacity, &tCreatedAt,
			&gID, &gFullName, &gAdditionals, &gConfirmAttendance, &gTableID, &gCreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// First row: initialize the table struct
		if result == nil {
			result = &types.TableAndGuests{
				ID: tID,

				Name:      tName,
				Capacity:  tCapacity,
				CreatedAt: tCreatedAt,
				Guests:    []types.Guest{},
			}
		}

		// Append guest if exists
		if gID.Valid {
			guest := types.Guest{
				ID:                int(gID.Int64),
				FullName:          gFullName.String,
				Additionals:       int(gAdditionals.Int64),
				ConfirmAttendance: gConfirmAttendance.Bool,
				CreatedAt:         gCreatedAt.Time,
			}
			if gTableID.Valid {
				id := int(gTableID.Int64)

				guest.TableId = &id
			}
			result.Guests = append(result.Guests, guest)
		}
	}

	if result == nil {
		return nil, fmt.Errorf("table with id %d not found", tableID)
	}

	return result, nil
}
