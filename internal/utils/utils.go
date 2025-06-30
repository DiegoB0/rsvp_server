package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

func ParseJSON(r *http.Request, payload any) error {
	if r.Body == nil {
		return fmt.Errorf("missing request body")
	}

	return json.NewDecoder(r.Body).Decode(payload)
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

// Standard error handler
func WriteError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, map[string]string{"error": err.Error()})
}

// Pagination helper functions
func NormalizePagination(p *types.PaginationParams) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
}

func Paginate[T any](
	db *sql.DB,
	baseQuery string,
	countQuery string,
	scanFunc func(*sql.Rows) (T, error),
	params types.PaginationParams,
	orderBy string,
	args ...interface{},
) (*types.PaginatedResult[T], error) {
	NormalizePagination(&params)

	offset := (params.Page - 1) * params.PageSize

	var totalCount int
	err := db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("error counting rows: %w", err)
	}

	limitPos := len(args) + 1
	offsetPos := len(args) + 2

	querySQL := fmt.Sprintf("%s ORDER BY %s LIMIT $%d OFFSET $%d", baseQuery, orderBy, limitPos, offsetPos)
	args = append(args, params.PageSize, offset)

	// query := fmt.Sprintf("%s LIMIT $1 OFFSET $2", baseQuery)
	// rows, err := db.Query(query, params.PageSize, offset)
	rows, err := db.Query(querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying rows: %w", err)
	}
	defer rows.Close()

	var result []T
	for rows.Next() {
		item, err := scanFunc(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	totalPages := (totalCount + params.PageSize - 1) / params.PageSize

	return &types.PaginatedResult[T]{
		Data:       result,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// Params to use in the HTTP requests
func ParsePaginationParams(r *http.Request) types.PaginationParams {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("page_size"))

	search := query.Get("search")
	var searchPtr *string
	if search != "" {
		searchPtr = &search
	}

	return types.PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Search:   searchPtr,
	}
}
