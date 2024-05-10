package models

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Idiom struct {
	ID           string           `db:"id" json:"id"`
	Idiom        string           `db:"idiom" json:"idiom"`
	MeaningBrief string           `db:"meaning_brief" json:"meaningBrief"`
	MeaningFull  string           `db:"meaning_full" json:"meaningFull"`
	CreatedAt    pgtype.Timestamp `db:"created_at" json:"createdAt"`
	PublishedAt  pgtype.Timestamp `db:"published_at" json:"publishedAt"`
	Thumbnail    pgtype.Text      `db:"thumbnail" json:"thumbnail"`
	Description  pgtype.Text      `db:"description" json:"description"`
	NumID        int64            `db:"num_id" json:"numId"`
	Examples     []string         `json:"examples"`
}

type IdiomDB struct {
	ID           string           `db:"id" json:"id"`
	Idiom        string           `db:"idiom" json:"idiom"`
	MeaningBrief string           `db:"meaning_brief" json:"meaningBrief"`
	MeaningFull  string           `db:"meaning_full" json:"meaningFull"`
	CreatedAt    pgtype.Timestamp `db:"created_at" json:"createdAt"`
	PublishedAt  pgtype.Timestamp `db:"published_at" json:"publishedAt"`
	Thumbnail    pgtype.Text      `db:"thumbnail" json:"thumbnail"`
	Description  pgtype.Text      `db:"description" json:"description"`
	NumID        int64            `db:"num_id" json:"numId"`
	Expression   string           `json:"expression" db:"expression"`
}

func (res *IdiomDB) ToIdiom() *Idiom {
	idiom := &Idiom{
		ID:           res.ID,
		Idiom:        res.Idiom,
		MeaningBrief: res.MeaningBrief,
		MeaningFull:  res.MeaningFull,
		CreatedAt:    res.CreatedAt,
		PublishedAt:  res.PublishedAt,
		Thumbnail:    res.Thumbnail,
		Description:  res.Description,
		NumID:        res.NumID,
		Examples:     []string{},
	}
	return idiom
}

type CursorToken struct {
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

type IdiomResponse struct {
	Idioms []Idiom     `json:"idioms"`
	Cursor CursorToken `json:"cursor"`
}

type IdiomThumbnailBody struct {
	ThumbnailPrompt string `json:"thumbnailPrompt"`
}

type IdiomInput struct {
	ID        string           `json:"id" db:"id"`
	Idiom     string           `json:"idiom" db:"idiom"`
	Meaning   string           `json:"meaning" db:"meaning"`
	CreatedAt pgtype.Timestamp `json:"createdAt" db:"created_at"`
}

type IdiomImageInput struct {
	Idiom  string `json:"idiom"`
	Prompt string `json:"prompt"`
}

type IdiomPrompt struct {
	Prompt string `json:"prompt"`
}

type IdiomDescription struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}
