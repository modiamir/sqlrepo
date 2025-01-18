package repository

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

type SampleEntity struct {
	Id   int64  `db:"id,autoincrement"`
	Name string `db:"name"`
}

func (e SampleEntity) GetID() int64 {
	return e.Id
}

func (e SampleEntity) GetTableName() string {
	return "sample_entities"
}

func (e SampleEntity) ToMap() map[string]interface{} {
	return make(map[string]interface{})
}

func InsertManyRecordsToSampleEntity(db *sql.DB, entities []SampleEntity) ([]int64, error) {
	var ids []int64
	for _, entity := range entities {
		id, err := InsertRecordsToSampleEntity(db, entity)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func InsertRecordsToSampleEntity(db *sql.DB, entity SampleEntity) (int64, error) {
	query := "INSERT INTO sample_entities (name) VALUES (?)"
	result, err := db.Exec(query, entity.Name)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func CreateSampleEntityTable(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sample_entities (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL
	)`)
	require.NoError(t, err)
}

func SelectSampleEntityByID(db *sql.DB, id int64) (SampleEntity, error) {
	var entity SampleEntity
	query := "SELECT * FROM sample_entities WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&entity.Id, &entity.Name)
	if err != nil {
		return SampleEntity{}, err
	}
	return entity, nil
}
