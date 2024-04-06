package idioms

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type PageToken [2]string

type Cursor struct {
	Idiom     *string           `json:"idiom"`
	CreatedAt *pgtype.Timestamp `json:"createdAt"`
}

type QueryFilter struct {
	OrderBy        string `json:"orderBy"`
	OrderDirection string `json:"orderDirection"`
	Keyword        string `json:"keyword"`
	Count          int    `json:"count"`

	operator string
	cursor   *Cursor
}
