package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type IntegrationTestSuite struct {
	suite.Suite
	MySQLContainer testcontainers.Container
	DB             *sql.DB
	Ctx            context.Context
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.Ctx = context.Background()
	port, err := nat.NewPort("tcp", "3306")
	s.Require().NoError(err)
	req := testcontainers.ContainerRequest{
		Name:         "sqlrepo_integration_test",
		Image:        "mysql:8.4",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "password",
			"MYSQL_DATABASE":      "sqlrepo_test",
		},
		WaitingFor: wait.ForSQL(port, "mysql", func(host string, port nat.Port) string {
			return "root:password@tcp(" + host + ":" + port.Port() + ")/"
		}),
	}

	s.MySQLContainer, err = testcontainers.GenericContainer(s.Ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            true,
	})

	mappedPort, _ := s.MySQLContainer.MappedPort(s.Ctx, port)

	_ = mappedPort

	s.Require().NoError(err)

	dbHost, err := s.MySQLContainer.Host(s.Ctx)
	s.Require().NoError(err)
	s.DB, err = sql.Open("mysql", "root:password@tcp("+dbHost+":"+mappedPort.Port()+")/sqlrepo_test")

	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) SetupTest() {
	// Get all tables and truncate them
	rows, err := s.DB.Query("SHOW TABLES")
	s.Require().NoError(err)
	defer rows.Close()

	var tableName string
	for rows.Next() {
		s.Require().NoError(rows.Scan(&tableName))
		_, err := s.DB.Exec("TRUNCATE TABLE " + tableName)
		s.Require().NoError(err)
		_, err = s.DB.Exec("DROP TABLE " + tableName)
		s.Require().NoError(err)
	}
}

func TestEntityRepository(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) TestNewEntityRepository() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	s.Assert().NotNil(repo)
}

func (s *IntegrationTestSuite) TestEntityRepository_FindAll() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entityId, err := InsertRecordsToSampleEntity(s.DB, SampleEntity{Name: "test"})
	s.Require().NoError(err)

	result, err := repo.FindAll()
	s.Assert().NoError(err)
	s.Assert().Len(result, 1)
	s.Assert().Equal(result[0].GetID(), entityId)
	s.Assert().Equal(result[0].Name, "test")
}

func (s *IntegrationTestSuite) TestEntityRepository_FindByID() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entityID, err := InsertRecordsToSampleEntity(s.DB, SampleEntity{Name: "test"})
	s.Require().NoError(err)

	result, err := repo.FindByID(entityID)
	s.Assert().NoError(err)
	s.Assert().Equal(result.GetID(), entityID)
	s.Assert().Equal(result.Name, "test")
}

func (s *IntegrationTestSuite) TestEntityRepository_FindAllByID() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	firstEntityID, err := InsertRecordsToSampleEntity(s.DB, SampleEntity{Name: "test"})
	s.Require().NoError(err)
	_, err = InsertRecordsToSampleEntity(s.DB, SampleEntity{Name: "test2"})
	s.Require().NoError(err)
	thirdEntityID, err := InsertRecordsToSampleEntity(s.DB, SampleEntity{Name: "test3"})
	s.Require().NoError(err)

	result, err := repo.FindAllByID([]int64{firstEntityID, thirdEntityID})
	s.Assert().NoError(err)
	s.Assert().Len(result, 2)
	s.Assert().Equal(result[0].GetID(), firstEntityID)
	s.Assert().Equal(result[0].Name, "test")
	s.Assert().Equal(result[1].GetID(), thirdEntityID)
	s.Assert().Equal(result[1].Name, "test3")
}

func (s *IntegrationTestSuite) TestEntityRepository_Save() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}

	err := repo.Save(&entity)
	s.Assert().NoError(err)

	sampleEntity, err := SelectSampleEntityByID(s.DB, entity.GetID())
	s.Require().NoError(err)

	s.Assert().NoError(err)
	s.Assert().Equal(entity.Name, sampleEntity.Name)
}

func (s *IntegrationTestSuite) TestEntityRepository_SaveAll() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}
	entityTwo := SampleEntity{Name: "test2"}

	err := repo.SaveAll([]*SampleEntity{&entity, &entityTwo})
	s.Assert().NoError(err)

	fetchedEntity, err := SelectSampleEntityByID(s.DB, entity.GetID())
	s.Assert().NoError(err)
	s.Assert().Equal(fetchedEntity.Name, entity.Name)

	fetchedEntityTwo, err := SelectSampleEntityByID(s.DB, entityTwo.GetID())
	s.Assert().NoError(err)
	s.Assert().Equal(fetchedEntityTwo.Name, entityTwo.Name)
}

func (s *IntegrationTestSuite) TestEntityRepository_DeleteAll() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}
	entityTwo := SampleEntity{Name: "test2"}

	_, err := InsertManyRecordsToSampleEntity(s.DB, []SampleEntity{entity, entityTwo})
	s.Require().NoError(err)

	err = repo.DeleteAll()
	s.Assert().NoError(err)

	result, err := repo.FindAll()
	s.Assert().NoError(err)
	s.Assert().Len(result, 0)
}

func (s *IntegrationTestSuite) TestEntityRepository_DeleteByIDs() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}
	entityTwo := SampleEntity{Name: "test2"}

	ids, err := InsertManyRecordsToSampleEntity(s.DB, []SampleEntity{entity, entityTwo})
	s.Require().NoError(err)

	err = repo.DeleteByIDs(ids)
	s.Assert().NoError(err)

	result, err := repo.FindAll()
	s.Assert().NoError(err)
	s.Assert().Len(result, 0)
}

func (s *IntegrationTestSuite) TestEntityRepository_DeleteByID() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}

	id, err := InsertRecordsToSampleEntity(s.DB, entity)
	s.Require().NoError(err)

	err = repo.DeleteByID(id)
	s.Assert().NoError(err)

	result, err := repo.FindAll()
	s.Assert().NoError(err)
	s.Assert().Len(result, 0)
}

func (s *IntegrationTestSuite) TestEntityRepository_DeleteEntities() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}
	entityTwo := SampleEntity{Name: "test2"}

	err := repo.SaveAll([]*SampleEntity{&entity, &entityTwo})
	s.Require().NoError(err)

	err = repo.DeleteEntities([]*SampleEntity{&entity, &entityTwo})
	s.Assert().NoError(err)

	result, err := repo.FindAll()
	s.Assert().NoError(err)
	s.Assert().Len(result, 0)
}

func (s *IntegrationTestSuite) TestEntityRepository_DeleteEntity() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}

	err := repo.Save(&entity)
	s.Require().NoError(err)

	err = repo.DeleteEntity(&entity)
	s.Assert().NoError(err)

	result, err := repo.FindAll()
	s.Assert().NoError(err)
	s.Assert().Len(result, 0)
}

func (s *IntegrationTestSuite) TestEntityRepository_ExistsByID() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}

	id, err := InsertRecordsToSampleEntity(s.DB, entity)
	s.Require().NoError(err)

	err = repo.ExistsByID(id)
	s.Assert().NoError(err)
}

func (s *IntegrationTestSuite) TestEntityRepository_FindAllPaginated() {
	repo := NewEntityRepository[SampleEntity](s.DB)
	CreateSampleEntityTable(s.T(), s.DB)
	entity := SampleEntity{Name: "test"}
	entityTwo := SampleEntity{Name: "test2"}

	_, err := InsertManyRecordsToSampleEntity(s.DB, []SampleEntity{entity, entityTwo})
	s.Require().NoError(err)

	result, err := repo.FindAllPaginated(Pagination{Limit: 1, Offset: 0})
	s.Assert().NoError(err)
	s.Assert().Len(result.Results, 1)
	s.Assert().Equal(result.TotalCount, 2)
	s.Assert().Equal(result.Results[0].Name, "test")

	result, err = repo.FindAllPaginated(Pagination{Limit: 1, Offset: 1})
	s.Assert().NoError(err)
	s.Assert().Len(result.Results, 1)
	s.Assert().Equal(result.TotalCount, 2)
	s.Assert().Equal(result.Results[0].Name, "test2")
}
