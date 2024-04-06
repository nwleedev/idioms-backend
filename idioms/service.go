package idioms

import (
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/nw.lee/idioms-backend/lib"
	"github.com/nw.lee/idioms-backend/logger"
	"github.com/nw.lee/idioms-backend/models"
)

type IdiomService interface {
	GetIdioms(cursor *QueryFilter, hasThumbnail bool) ([]models.Idiom, error)
	GetIdiomById(id string) (*models.Idiom, error)
	SearchIdioms(cursor *QueryFilter, hasThumbnail bool) ([]models.Idiom, error)
	GetRelatedIdioms(idiomId string) ([]models.Idiom, error)
	UpdateThumbnailPrompt(idiomId string, newPrompt string) (*string, error)
	CreateIdiomInputs(inputs []models.IdiomInput) (*int, error)
}

type Service struct {
	db     *sqlx.DB
	logger logger.LoggerService
}

func NewService(db *sqlx.DB, logger logger.LoggerService) *Service {
	service := new(Service)
	service.db = db
	service.logger = logger
	return service
}

func (service *Service) GetIdiomById(id string) (*models.Idiom, error) {
	var idioms = []models.IdiomDB{}
	var idiom *models.Idiom
	var examples []string

	sql, _, _ := sq.
		Select("idioms.*, examples.expression as expression").
		From("idioms").
		Where("idioms.id = $1", id).
		Join("idiom_examples as examples on idioms.id = examples.idiom_id").
		ToSql()
	err := service.db.Select(&idioms, sql, id)
	if err != nil || len(idioms) == 0 {
		service.logger.Println("Cannot find a idiom by %s", id)
		return nil, err
	}

	for index := range idioms {
		if index == 0 {
			idiom = idioms[index].ToIdiom()
		}
		examples = append(examples, idioms[index].Expression)
	}
	idiom.Examples = examples
	return idiom, nil
}

func (service *Service) GetIdioms(filter *QueryFilter, hasThumbnail bool) ([]models.Idiom, error) {
	idiomResponses := []models.IdiomDB{}
	idioms := []models.Idiom{}
	query := sq.Select("*").From("idioms").OrderBy(fmt.Sprintf("%s %s", filter.OrderBy, filter.OrderDirection)).Limit(uint64(filter.Count))
	if filter.cursor.Idiom != nil {
		query = query.Where(fmt.Sprintf("%s %s $1", filter.OrderBy, filter.operator), *filter.cursor.Idiom)
	}
	if filter.cursor.CreatedAt != nil {
		createdAt := filter.cursor.CreatedAt.Time.Format(time.RFC3339Nano)
		query = query.Where(fmt.Sprintf("%s %s $1", filter.OrderBy, filter.operator), createdAt)
	}
	if hasThumbnail {
		query = query.Where("thumbnail IS NOT NULL")
	}
	sql, args, _ := query.ToSql()
	err := service.db.Select(&idiomResponses, sql, args...)

	if err != nil {
		service.logger.Println("Cannot find idioms")
		return nil, err
	}

	for _, response := range idiomResponses {
		idioms = append(idioms, *response.ToIdiom())
	}
	return idioms, nil
}

func (service *Service) SearchIdioms(filter *QueryFilter, hasThumbnail bool) ([]models.Idiom, error) {
	idiomResponses := []models.IdiomDB{}
	idioms := []models.Idiom{}
	query := sq.Select("*").From("idioms").OrderBy(fmt.Sprintf("%s %s", filter.OrderBy, filter.OrderDirection)).Limit(uint64(filter.Count))
	if filter.cursor.Idiom != nil {
		query = query.Where(fmt.Sprintf("%s %s ?", filter.OrderBy, filter.operator), *filter.cursor.Idiom)
	}
	if filter.cursor.CreatedAt != nil {
		createdAt := filter.cursor.CreatedAt.Time.Format(time.RFC3339Nano)
		query = query.Where(fmt.Sprintf("%s %s ?", filter.OrderBy, filter.operator), createdAt)
	}
	if hasThumbnail {
		query = query.Where("thumbnail IS NOT NULL")
	}
	keywords := strings.Split(filter.Keyword, " ")
	likes := sq.Or{}
	for _, keyword := range keywords {
		likes = sq.Or{
			likes,
			sq.Expr("idiom ilike ?", fmt.Sprintf("%%%s%%", keyword)),
		}
	}
	query = query.Where(likes)

	sql, args, _ := query.PlaceholderFormat(sq.Dollar).ToSql()
	err := service.db.Select(&idiomResponses, sql, args...)

	if err != nil {
		service.logger.PrintError("Query error", err)
		return nil, err
	}

	for _, response := range idiomResponses {
		idioms = append(idioms, *response.ToIdiom())
	}
	return idioms, nil
}

func (service *Service) GetRelatedIdioms(idiomId string) ([]models.Idiom, error) {
	ascQuery, _, _ := sq.Select("idioms.*").From("idioms as idioms").Join("idioms as target on target.id = $1").Where("idioms.created_at > target.created_at").Where("idioms.thumbnail is not null").OrderBy("idioms.created_at asc").Limit(4).PlaceholderFormat(sq.Dollar).ToSql()
	descQuery, _, _ := sq.Select("idioms.*").From("idioms as idioms").Join("idioms as target on target.id = $2").Where("idioms.created_at < target.created_at").Where("idioms.thumbnail is not null").OrderBy("idioms.created_at desc").Limit(4).PlaceholderFormat(sq.Dollar).ToSql()

	// SQL without any parameters
	fromStatement := fmt.Sprintf("((%s) union (%s)) as related", ascQuery, descQuery)
	query, _, err := sq.Select("related.*").From(fromStatement).OrderBy("related.created_at desc").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Println("Failed to create a query with id: %s", idiomId)
		service.logger.PrintError("Error: %v", err)
		return nil, err
	}

	idiomResponses := []models.IdiomDB{}
	idioms := []models.Idiom{}
	err = service.db.Select(&idiomResponses, query, idiomId, idiomId)
	if err != nil {
		service.logger.Println("Failed to query the related idioms with id: %s", idiomId)
		service.logger.PrintError("Error: %v", err)
		return nil, err
	}

	for _, response := range idiomResponses {
		idioms = append(idioms, *response.ToIdiom())
	}

	return idioms, err
}

func (service *Service) UpdateThumbnailPrompt(idiomId string, newPrompt string) (*string, error) {
	query, args, err := sq.Update("idioms").Set("thumbnail_prompt", newPrompt).Where("id = ?", idiomId).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Println("Failed to query with id %s", idiomId)
		return nil, err
	}

	_, err = service.db.Exec(query, args...)
	if err != nil {
		service.logger.Println("Failed to update prompt with id %s", idiomId)
		return nil, err
	}

	return &newPrompt, nil
}

func (service *Service) CreateIdiomInputs(inputs []models.IdiomInput) (*int, error) {
	query := sq.Insert("idiom_inputs").Columns("id", "idiom", "meaning")
	for _, input := range inputs {
		query = query.Values(lib.ToIdiomID(input.Idiom), input.Idiom, input.Meaning)
	}
	sql, args, err := query.Suffix("on conflict (id) do nothing").PlaceholderFormat(sq.Dollar).ToSql()

	if err != nil {
		service.logger.Println("Failed to make query with following inputs")
		service.logger.PrintError("", err)
		return nil, err
	}

	result, err := service.db.Exec(sql, args...)

	if err != nil {
		service.logger.Println("Failed to create idiom inputs with following inputs")
		service.logger.PrintError("", err)
		return nil, err
	}

	affected, _ := result.RowsAffected()
	rows := int(affected)
	return &rows, nil
}
