package repository

type Entity[ID comparable] interface {
	GetID() ID
	GetTableName() string
	ToMap() map[string]any
}

type Repository[E Entity[ID], ID comparable] interface {
	FindAll() ([]*E, error)
	FindAllByID(ids []ID) ([]*E, error)
	FindByID(id ID) (*E, error)
	Save(*E) error
	SaveAll(entities []*E) error
	DeleteByID(ID) error
	DeleteByIDs([]ID) error
	DeleteAll() error
	DeleteEntities(entities []*E) error
	DeleteEntity(entity *E) error
	ExistsByID(id ID) error
	FindAllPaginated(pagination Pagination) (*PaginatedResult[E], error)
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PaginatedResult[E any] struct {
	Pagination Pagination `json:"pagination"`
	TotalCount int        `json:"total_count"`
	Results    []*E       `json:"results"`
}
