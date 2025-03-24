package storage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/certainty/oneapi/internal/spec"
	"slices"
)

type FieldValidator func(any) error

// Entity represents a defined entity with validation rules
type Entity struct {
	Name        string
	Fields      map[string]Field
	Validations map[string][]FieldValidator
}

// Field represents a field of an entity
type Field struct {
	Name     string
	Type     string
	Required bool
	Variants []string
}

func NewEntity(name string, def spec.EntityDef) *Entity {
	entity := &Entity{
		Name:        name,
		Fields:      make(map[string]Field),
		Validations: make(map[string][]FieldValidator),
	}

	for name, field := range def.Fields {
		entity.Fields[name] = Field{
			Type:     field.Type,
			Required: field.Required,
			Variants: field.Variants,
		}

		// Create validations for the field
		var validations []FieldValidator

		// Required validation
		if field.Required {
			validations = append(validations, func(value any) error {
				if value == nil {
					return fmt.Errorf("field %s is required", name)
				}
				if s, ok := value.(string); ok && strings.TrimSpace(s) == "" {
					return fmt.Errorf("field %s is required", name)
				}
				return nil
			})
		}

		// Type-specific validations
		switch field.Type {
		case "string":
			validations = append(validations, func(value any) error {
				if value == nil {
					return nil
				}
				if _, ok := value.(string); !ok {
					return fmt.Errorf("field %s must be a string", name)
				}
				return nil
			})
		case "int":
			validations = append(validations, func(value any) error {
				if value == nil {
					return nil
				}

				switch v := value.(type) {
				case int, int64:
					return nil
				case string:
					if _, err := strconv.ParseInt(v, 10, 64); err != nil {
						return fmt.Errorf("field %s must be a number", name)
					}
				default:
					return fmt.Errorf("field %s must be an integer", name)
				}
				return nil
			})
		case "double":
			validations = append(validations, func(value any) error {
				if value == nil {
					return nil
				}
				switch v := value.(type) {
				case float64:
					return nil
				case string:
					if _, err := strconv.ParseFloat(v, 64); err != nil {
						return fmt.Errorf("field %s must be a number", name)
					}
				default:
					return fmt.Errorf("field %s must be a number", name)
				}
				return nil
			})
		case "enum":
			variants := field.Variants
			validations = append(validations, func(value any) error {
				if value == nil {
					return nil
				}
				strValue, ok := value.(string)
				if !ok {
					return fmt.Errorf("field %s must be a string for enum type", name)
				}

				valid := slices.Contains(variants, strValue)
				if !valid {
					return fmt.Errorf("field %s must be one of: %s", name, strings.Join(variants, ", "))
				}
				return nil
			})
		}

		entity.Validations[name] = validations
	}

	return entity
}

func (e *Entity) GetFieldType(fieldName string) string {
	field, exists := e.Fields[fieldName]
	if !exists {
		return ""
	}

	switch field.Type {
	case "string", "enum":
		return "TEXT"
	case "int":
		return "INTEGER"
	case "double":
		return "REAL"
	case "bool":
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

func (e *Entity) Validate(data map[string]any) []error {
	var errs []error

	// Check for unknown fields
	for fieldName := range data {
		if _, exists := e.Fields[fieldName]; !exists {
			errs = append(errs, fmt.Errorf("unknown field: %s", fieldName))
		}
	}

	// Validate each field
	for fieldName, validators := range e.Validations {
		value, exists := data[fieldName]

		// If field doesn't exist but is required, it will be caught by required validator
		if !exists {
			value = nil
		}

		for _, validator := range validators {
			if err := validator(value); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

type EntityRegistry struct {
	entities map[string]*Entity
}

// NewEntityRegistry creates a new entity registry
func NewEntityRegistry() *EntityRegistry {
	return &EntityRegistry{
		entities: make(map[string]*Entity),
	}
}

// RegisterEntity registers an entity definition
func (r *EntityRegistry) RegisterEntity(name string, def spec.EntityDef) *Entity {
	entity := NewEntity(name, def)
	r.entities[name] = entity
	return entity
}

// GetEntity retrieves an entity by name
func (r *EntityRegistry) GetEntity(name string) (*Entity, error) {
	entity, exists := r.entities[name]
	if !exists {
		return nil, fmt.Errorf("entity %s not found", name)
	}
	return entity, nil
}
