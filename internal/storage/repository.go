package storage

import (
	"database/sql"
	"fmt"
	"maps"
	"strings"
)

type Repository interface {
	CreateSchema() error
	List(page, pageSize int) ([]map[string]any, int, error)
	FindByID(id int64) (map[string]any, error)
	Create(data map[string]any) (int64, error)
	Update(id int64, data map[string]any) error
	Delete(id int64) error
}

// SQLiteRepository implements Repository for SQLite
type SQLiteRepository struct {
	db     *DB
	entity *Entity
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(db *DB, entity *Entity) *SQLiteRepository {
	return &SQLiteRepository{
		db:     db,
		entity: entity,
	}
}

func (r *SQLiteRepository) CreateSchema() error {
	var columns []string
	columns = append(columns, "id INTEGER PRIMARY KEY AUTOINCREMENT")

	for fieldName, field := range r.entity.Fields {
		sqlType := r.entity.GetFieldType(fieldName)
		if sqlType == "" {
			continue
		}

		column := fmt.Sprintf("%s %s", fieldName, sqlType)
		if field.Required {
			column += " NOT NULL"
		}
		columns = append(columns, column)
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)",
		r.entity.Name, strings.Join(columns, ", "))

	return r.db.Exec(query)
}

func (r *SQLiteRepository) List(page, pageSize int) ([]map[string]any, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// Count total rows
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.entity.Name)
	err := r.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch paginated data
	query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", r.entity.Name)
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}

	result := make([]map[string]any, 0)
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, 0, err
		}

		// Create a map to hold the row data
		entry := make(map[string]any)
		for i, col := range columns {
			val := values[i]

			// Handle nil values
			if val == nil {
				entry[col] = nil
				continue
			}

			// Convert to appropriate type
			switch v := val.(type) {
			case []byte:
				entry[col] = string(v)
			default:
				entry[col] = v
			}
		}

		result = append(result, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

// FindByID retrieves an entity by ID
func (r *SQLiteRepository) FindByID(id int64) (map[string]any, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", r.entity.Name)
	row := r.db.QueryRow(query, id)

	// Get columns
	rows, err := r.db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 1", r.entity.Name))
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	rows.Close()
	if err != nil {
		return nil, err
	}

	// Create a slice of interface{} to hold the values
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity with id %d not found", id)
		}
		return nil, err
	}

	// Create a map to hold the row data
	result := make(map[string]any)
	for i, col := range columns {
		val := values[i]

		// Handle nil values
		if val == nil {
			result[col] = nil
			continue
		}

		// Convert to appropriate type
		switch v := val.(type) {
		case []byte:
			result[col] = string(v)
		default:
			result[col] = v
		}
	}

	return result, nil
}

// Create creates a new entity
func (r *SQLiteRepository) Create(data map[string]any) (int64, error) {
	// Validate data
	errs := r.entity.Validate(data)
	if len(errs) > 0 {
		return 0, fmt.Errorf("validation failed: %v", errs)
	}

	// Prepare SQL fields and values
	var fields []string
	var placeholders []string
	var values []any

	for fieldName, value := range data {
		// Skip if the field doesn't exist in the entity
		if _, exists := r.entity.Fields[fieldName]; !exists {
			continue
		}

		fields = append(fields, fieldName)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}

	// Execute query
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		r.entity.Name, strings.Join(fields, ", "), strings.Join(placeholders, ", "))

	result, err := r.db.DB.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Update updates an existing entity
func (r *SQLiteRepository) Update(id int64, data map[string]any) error {
	// Check if entity exists
	_, err := r.FindByID(id)
	if err != nil {
		return err
	}

	// Get existing data to merge with update data
	existing, err := r.FindByID(id)
	if err != nil {
		return err
	}

	// Merge data for validation
	merged := make(map[string]any)
	for k, v := range existing {
		if k != "id" { // Skip the ID field
			merged[k] = v
		}
	}
	maps.Copy(merged, data)

	// Validate merged data
	errs := r.entity.Validate(merged)
	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %v", errs)
	}

	// Prepare SQL fields and values for update
	var setClause []string
	var values []any

	for fieldName, value := range data {
		// Skip if the field doesn't exist in the entity
		if _, exists := r.entity.Fields[fieldName]; !exists {
			continue
		}

		setClause = append(setClause, fmt.Sprintf("%s = ?", fieldName))
		values = append(values, value)
	}

	// Add ID for the WHERE clause
	values = append(values, id)

	// Execute query
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
		r.entity.Name, strings.Join(setClause, ", "))

	_, err = r.db.DB.Exec(query, values...)
	return err
}

// Delete deletes an entity
func (r *SQLiteRepository) Delete(id int64) error {
	// Check if entity exists
	_, err := r.FindByID(id)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", r.entity.Name)
	_, err = r.db.DB.Exec(query, id)
	return err
}
