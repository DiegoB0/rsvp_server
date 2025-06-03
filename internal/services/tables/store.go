package tables

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
