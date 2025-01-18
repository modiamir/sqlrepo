package repository

import (
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
)

func NewEntityRepository[E Entity[ID], ID comparable](db *sql.DB) Repository[E, ID] {
	return &entityRepository[E, ID]{
		DB: sqlx.NewDb(db, "mysql"),
	}
}

type entityRepository[E Entity[ID], ID comparable] struct {
	DB *sqlx.DB
}

func (r *entityRepository[E, ID]) FindAll() ([]*E, error) {
	var emptyEntity E
	tableName := emptyEntity.GetTableName()

	var entities []*E
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	err := r.DB.Select(&entities, query)
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *entityRepository[E, ID]) FindByID(id ID) (*E, error) {
	entities, err := r.FindAllByID([]ID{id})
	if err != nil {
		return nil, err
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("entity not found")
	}

	return entities[0], nil
}

func (r *entityRepository[E, ID]) FindAllByID(ids []ID) ([]*E, error) {
	var emptyEntity E
	tableName := emptyEntity.GetTableName()
	args := make([]interface{}, len(ids))
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = fmt.Sprintf("?")
		args[i] = id
	}

	var entities []*E
	query := fmt.Sprintf("SELECT * FROM %s WHERE id IN (%s)", tableName, strings.Join(idStrings, ","))
	err := r.DB.Select(&entities, query, args...)
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *entityRepository[E, ID]) Save(entity *E) error {
	return r.SaveAll([]*E{entity})
}

func (r *entityRepository[E, ID]) SaveAll(entities []*E) error {
	if len(entities) == 0 {
		return nil
	}

	var columns []string
	var placeholders []string
	var values []interface{}

	// Use the first entity to determine the columns
	firstEntity := entities[0]
	entityValue := reflect.ValueOf(firstEntity).Elem()
	entityType := entityValue.Type()

	// Ensure entity implements Entity interface
	entityInterface, ok := any(firstEntity).(Entity[ID])
	if !ok {
		return fmt.Errorf("entity does not implement the Entity interface")
	}

	var idAutoIncrement bool
	var idField reflect.StructField

	// Iterate over the fields of the struct
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		dbTag := field.Tag.Get("db")
		tagParts := strings.Split(dbTag, ",")
		for j, tagPart := range tagParts {
			tagParts[j] = strings.TrimSpace(tagPart)
		}
		if len(tagParts) > 0 {
			columnName := tagParts[0]
			if columnName == "id" {
				idAutoIncrement = len(tagParts) > 1 && slices.Contains(tagParts, "autoincrement")
				idField = field

				if idAutoIncrement {
					continue
				}
			}
			columns = append(columns, columnName)
			placeholders = append(placeholders, "?")
		}
	}

	// Build the query
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", entityInterface.GetTableName(), strings.Join(columns, ","))

	// Add placeholders and values for each entity
	for _, entity := range entities {
		entityValue := reflect.ValueOf(entity).Elem()
		for i := 0; i < entityType.NumField(); i++ {
			field := entityType.Field(i)
			dbTag := field.Tag.Get("db")
			tagParts := strings.Split(dbTag, ",")
			columnName := strings.TrimSpace(tagParts[0])
			if columnName == "id" && idAutoIncrement {
				continue
			}
			values = append(values, entityValue.Field(i).Interface())
		}
		query += fmt.Sprintf("(%s),", strings.Join(placeholders, ","))
	}

	// Remove the trailing comma
	query = strings.TrimSuffix(query, ",")

	// Execute the query
	result, err := r.DB.Exec(query, values...)
	if err != nil {
		return err
	}

	// Set auto-increment IDs if necessary
	if idAutoIncrement {
		lastInsertID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		for i, entity := range entities {
			entityValue := reflect.ValueOf(entity).Elem()
			entityValue.FieldByName(idField.Name).SetInt(lastInsertID + int64(i))
		}
	}

	return nil
}

func (r *entityRepository[E, ID]) DeleteByID(id ID) error {
	return r.DeleteByIDs([]ID{id})
}

func (r *entityRepository[E, ID]) DeleteByIDs(ids []ID) error {
	var emptyEntity E
	tableName := emptyEntity.GetTableName()
	args := make([]interface{}, len(ids))
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = fmt.Sprintf("?")
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", tableName, strings.Join(idStrings, ","))
	_, err := r.DB.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *entityRepository[E, ID]) DeleteAll() error {
	var emptyEntity E
	tableName := emptyEntity.GetTableName()
	query := fmt.Sprintf("DELETE FROM %s", tableName)
	_, err := r.DB.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (r *entityRepository[E, ID]) DeleteEntities(entities []*E) error {
	var ids []ID
	for _, entity := range entities {
		entityInterface, ok := any(entity).(Entity[ID])
		if !ok {
			return fmt.Errorf("entity does not implement the Entity interface")
		}
		ids = append(ids, entityInterface.GetID())
	}
	return r.DeleteByIDs(ids)
}

func (r *entityRepository[E, ID]) DeleteEntity(entity *E) error {
	return r.DeleteEntities([]*E{entity})
}

func (r *entityRepository[E, ID]) ExistsByID(id ID) error {
	entities, err := r.FindAllByID([]ID{id})
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

func (r *entityRepository[E, ID]) FindAllPaginated(pagination Pagination) (*PaginatedResult[E], error) {
	var emptyEntity E
	tableName := emptyEntity.GetTableName()

	var entities []*E
	query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", tableName)
	err := r.DB.Select(&entities, query, pagination.Limit, pagination.Offset)
	if err != nil {
		return nil, err
	}

	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err = r.DB.Get(&totalCount, countQuery)
	if err != nil {
		return nil, err
	}

	return &PaginatedResult[E]{
		Pagination: pagination,
		TotalCount: totalCount,
		Results:    entities,
	}, nil
}
