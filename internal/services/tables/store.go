package tables

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/diegob0/rspv_backend/internal/utils"
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

func scanRowIntoTableAndGuests(rows *sql.Rows) (*types.TableAndGuests, error) {
	t := new(types.TableAndGuests)

	err := rows.Scan(
		&t.ID,
		&t.Name,
		&t.Capacity,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.Guests = []types.Guest{}
	t.Generals = []types.General{}

	return t, nil
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

func (s *Store) GetTables(params types.PaginationParams) (*types.PaginatedResult[*types.Table], error) {
	var whereClause string
	var args []interface{}
	orderBy := "id"

	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		whereClause = " WHERE name ILIKE $1"
		args = append(args, "%"+strings.TrimSpace(*params.Search)+"%")
	}

	baseQuery := `
		SELECT id, name, capacity, created_at::timestamptz
		FROM tables
	` + whereClause

	countQuery := `SELECT COUNT(*) FROM tables` + whereClause

	return utils.Paginate(s.db, baseQuery, countQuery, scanRowIntoTable, params, orderBy, args...)
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

func (s *Store) GetTablesWithGuests(params types.PaginationParams) (*types.PaginatedResult[*types.TableAndGuests], error) {
	var whereClause string
	var args []interface{}
	orderBy := "id"

	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		whereClause = " WHERE name ILIKE $1"
		args = append(args, "%"+strings.TrimSpace(*params.Search)+"%")
	}

	baseQuery := `
		SELECT id, name, capacity, created_at::timestamptz
		FROM tables
	` + whereClause

	countQuery := `SELECT COUNT(*) FROM tables` + whereClause

	// Step 1: Paginate the base table info
	paginated, err := utils.Paginate(s.db, baseQuery, countQuery, scanRowIntoTableAndGuests, params, orderBy, args...)
	if err != nil {
		return nil, err
	}

	// Step 2: Build map of tableID to table reference
	tablesMap := make(map[int]*types.TableAndGuests)
	for _, table := range paginated.Data {
		tablesMap[table.ID] = table
	}

	// Step 3: Fetch and attach guests
	guestRows, err := s.db.Query(`
		SELECT id, full_name, additionals, confirm_attendance, table_id, created_at::timestamptz
		FROM guests
		WHERE table_id IS NOT NULL
		ORDER BY table_id, id

	`)
	if err != nil {
		return nil, err
	}
	defer guestRows.Close()

	for guestRows.Next() {
		var g types.Guest
		var tableID int
		err := guestRows.Scan(&g.ID, &g.FullName, &g.Additionals, &g.ConfirmAttendance, &tableID, &g.CreatedAt)
		if err != nil {
			return nil, err
		}
		g.TableId = &tableID
		if t, ok := tablesMap[tableID]; ok {
			t.Guests = append(t.Guests, g)
		}
	}

	// Step 4: Fetch and attach generals
	genRows, err := s.db.Query(`
		SELECT id, folio, table_id, qr_code_url, pdf_file, created_at::timestamptz
		FROM generals
		WHERE table_id IS NOT NULL
		ORDER BY table_id, id
	`)
	if err != nil {
		return nil, err
	}
	defer genRows.Close()

	for genRows.Next() {
		var gen types.General
		var tableID int
		err := genRows.Scan(&gen.ID, &gen.Folio, &tableID, &gen.QrCodeUrl, &gen.PDFUrl, &gen.CreatedAt)
		if err != nil {
			return nil, err
		}

		gen.TableId = &tableID

		if t, ok := tablesMap[tableID]; ok {
			t.Generals = append(t.Generals, gen)
		}
	}

	return paginated, nil
}

func (s *Store) GetTableWithGuestsByID(tableID int) (*types.TableAndGuests, error) {
	var table types.TableAndGuests
	tableQuery := `
		SELECT id, name, capacity, created_at::timestamptz
		FROM tables
		WHERE id = $1;
	`
	err := s.db.QueryRow(tableQuery, tableID).Scan(&table.ID, &table.Name, &table.Capacity, &table.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("table with id %d not found", tableID)
		}
		return nil, err
	}

	table.Guests = []types.Guest{}

	table.Generals = []types.General{}

	guestsQuery := `
		SELECT id, full_name, additionals, confirm_attendance, table_id, created_at::timestamptz
		FROM guests
		WHERE table_id = $1
		ORDER BY id;
	`
	guestRows, err := s.db.Query(guestsQuery, tableID)
	if err != nil {
		return nil, err
	}
	defer guestRows.Close()

	for guestRows.Next() {
		var g types.Guest
		var tID int
		err := guestRows.Scan(&g.ID, &g.FullName, &g.Additionals, &g.ConfirmAttendance, &tID, &g.CreatedAt)
		if err != nil {
			return nil, err
		}
		g.TableId = &tID

		table.Guests = append(table.Guests, g)
	}

	generalsQuery := `
		SELECT id, folio, table_id, qr_code_url, pdf_file, created_at::timestamptz
		FROM generals
		WHERE table_id = $1
		ORDER BY id;
	`
	genRows, err := s.db.Query(generalsQuery, tableID)
	if err != nil {
		return nil, err
	}

	defer genRows.Close()

	for genRows.Next() {
		var gen types.General

		var tID int
		err := genRows.Scan(&gen.ID, &gen.Folio, &tID, &gen.QrCodeUrl, &gen.PDFUrl, &gen.CreatedAt)
		if err != nil {
			return nil, err
		}
		gen.TableId = &tID
		table.Generals = append(table.Generals, gen)
	}

	return &table, nil
}
