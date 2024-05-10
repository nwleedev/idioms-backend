package idioms

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/nw.lee/idioms-backend/lib"
	"github.com/nw.lee/idioms-backend/logger"
	"github.com/nw.lee/idioms-backend/models"
	"github.com/nw.lee/idioms-backend/openai"
)

type IdiomService interface {
	GetMainPageIdioms() ([]models.Idiom, error)
	GetIdioms(cursor *QueryFilter, hasThumbnail bool) ([]models.Idiom, error)
	GetIdiomById(id string) (*models.Idiom, error)
	SearchIdioms(cursor *QueryFilter, hasThumbnail bool) ([]models.Idiom, error)
	GetRelatedIdioms(idiomId string) ([]models.Idiom, error)
	UpdateThumbnailPrompt(idiomId string, newPrompt string) (*string, error)
	CreateIdiomInputs(inputs []models.IdiomInput) (*int, error)
	CreateDescription(id string) (*models.IdiomDescription, error)
}

type Service struct {
	db     *sqlx.DB
	logger logger.LoggerService
	ai     openai.OpenAiInterface
}

func NewService(db *sqlx.DB, logger logger.LoggerService, ai openai.OpenAiInterface) *Service {
	service := new(Service)
	service.db = db
	service.logger = logger
	service.ai = ai
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
		service.logger.Warn("Cannot find a idiom by", id)
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

	innerQueryBuilder := sq.Select("*").From("idioms").Limit(uint64(filter.Count))
	if filter.idiom == nil && filter.createdAt == nil {
		innerOrderBy := fmt.Sprintf("%s %s", filter.OrderBy, filter.OrderDirection)
		innerQueryBuilder = innerQueryBuilder.OrderBy(innerOrderBy)
	} else {
		innerOrderBy := fmt.Sprintf("%s %s", filter.OrderBy, filter.innerOrderDirection)
		innerQueryBuilder = innerQueryBuilder.OrderBy(innerOrderBy)
	}

	if filter.idiom != nil {
		innerWhere := fmt.Sprintf("%s %s ?", filter.OrderBy, filter.operator)
		innerQueryBuilder = innerQueryBuilder.Where(innerWhere, *filter.idiom)
	}
	if filter.createdAt != nil {
		createdAt := filter.createdAt.Time.Format(time.RFC3339Nano)

		innerWhere := fmt.Sprintf("%s %s $1", filter.OrderBy, filter.operator)
		innerQueryBuilder = innerQueryBuilder.Where(innerWhere, createdAt)
	}
	if hasThumbnail {
		innerQueryBuilder = innerQueryBuilder.Where("thumbnail IS NOT NULL")
	}
	innerQuery, innerArgs, err := innerQueryBuilder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Error(err, "Failed to create a query", filter)
		return nil, err
	}
	join := fmt.Sprintf("(%s) as source on source.id = target.id", innerQuery)
	orderBy := fmt.Sprintf("%s %s", filter.OrderBy, filter.OrderDirection)
	query, _, err := sq.Select("target.*").From("idioms as target").Join(join).OrderBy(orderBy).ToSql()

	if err != nil {
		service.logger.Error(err, "Failed to join a queries", filter)
		return nil, err
	}
	err = service.db.Select(&idiomResponses, query, innerArgs...)
	if err != nil {
		service.logger.Warn("Cannot find idioms")
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

	innerBuilder := sq.Select("*").From("idioms").Limit(uint64(filter.Count))
	if filter.idiom == nil && filter.createdAt == nil {
		innerOrderBy := fmt.Sprintf("%s %s", filter.OrderBy, filter.OrderDirection)
		innerBuilder = innerBuilder.OrderBy(innerOrderBy)
	} else {
		innerOrderBy := fmt.Sprintf("%s %s", filter.OrderBy, filter.innerOrderDirection)
		innerBuilder = innerBuilder.OrderBy(innerOrderBy)
	}

	if filter.idiom != nil {
		innerBuilder = innerBuilder.Where(fmt.Sprintf("%s %s ?", filter.OrderBy, filter.operator), *filter.idiom)
	}
	if filter.createdAt != nil {
		createdAt := filter.createdAt.Time.Format(time.RFC3339Nano)
		innerBuilder = innerBuilder.Where(fmt.Sprintf("%s %s ?", filter.OrderBy, filter.operator), createdAt)
	}
	if hasThumbnail {
		innerBuilder = innerBuilder.Where("thumbnail IS NOT NULL")
	}
	keywords := strings.Split(filter.Keyword, " ")
	likes := sq.Or{}
	for _, keyword := range keywords {
		if len(keyword) < 2 {
			continue
		}
		likes = sq.Or{
			likes,
			sq.Expr("idiom ilike ?", fmt.Sprintf("%%%s%%", keyword)),
		}
	}
	innerBuilder = innerBuilder.Where(likes)
	innerQuery, innerArgs, err := innerBuilder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Error(err, "Failed to create a query", filter)
		return nil, err
	}
	join := fmt.Sprintf("(%s) as source on source.id = target.id", innerQuery)
	orderBy := fmt.Sprintf("%s %s", filter.OrderBy, filter.OrderDirection)
	query, _, err := sq.Select("target.*").From("idioms as target").Join(join).OrderBy(orderBy).ToSql()

	if err != nil {
		service.logger.Error(err, "Failed to join a queries", filter)
		return nil, err
	}
	err = service.db.Select(&idiomResponses, query, innerArgs...)
	if err != nil {
		service.logger.Warn("Cannot find idioms")
		return nil, err
	}

	for _, response := range idiomResponses {
		idioms = append(idioms, *response.ToIdiom())
	}
	return idioms, nil
}

func (service *Service) GetRelatedIdioms(idiomId string) ([]models.Idiom, error) {
	ascQuery, _, _ := sq.Select("idioms.*").From("idioms as idioms").Join("idioms as target on target.id = $1").Where("idioms.published_at > target.published_at").Where("idioms.thumbnail is not null").OrderBy("idioms.published_at asc").Limit(4).PlaceholderFormat(sq.Dollar).ToSql()
	descQuery, _, _ := sq.Select("idioms.*").From("idioms as idioms").Join("idioms as target on target.id = $2").Where("idioms.published_at < target.published_at").Where("idioms.thumbnail is not null").OrderBy("idioms.published_at desc").Limit(4).PlaceholderFormat(sq.Dollar).ToSql()

	// SQL without any parameters
	fromStatement := fmt.Sprintf("((%s) union (%s)) as related", ascQuery, descQuery)
	query, _, err := sq.Select("related.*").From(fromStatement).OrderBy("related.published_at desc").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Error(err, "Failed to create a query with id", idiomId)
		return nil, err
	}

	idiomResponses := []models.IdiomDB{}
	idioms := []models.Idiom{}
	err = service.db.Select(&idiomResponses, query, idiomId, idiomId)
	if err != nil {
		service.logger.Error(err, "Failed to query the related idioms with id", idiomId)
		return nil, err
	}

	for _, response := range idiomResponses {
		idioms = append(idioms, *response.ToIdiom())
	}

	return idioms, err
}

func (service *Service) GetMainPageIdioms() ([]models.Idiom, error) {
	query, args, err := sq.Select("*").From("idioms").Limit(24).OrderBy("published_at desc").Where("thumbnail is not null").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Error(err, "Failed to create a query.")
		return nil, err
	}
	idiomResponses := []models.IdiomDB{}
	idioms := []models.Idiom{}
	err = service.db.Select(&idiomResponses, query, args...)
	if err != nil {
		service.logger.Error(err, "Failed to query idioms from db.")
		return nil, err
	}

	for _, response := range idiomResponses {
		idioms = append(idioms, *response.ToIdiom())
	}
	return idioms, nil
}

func (service *Service) UpdateThumbnailPrompt(idiomId string, newPrompt string) (*string, error) {
	query, args, err := sq.Update("idioms").Set("thumbnail_prompt", newPrompt).Where("id = ?", idiomId).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Error(err, "Failed to query with id", idiomId)
		return nil, err
	}

	_, err = service.db.Exec(query, args...)
	if err != nil {
		service.logger.Error(err, "Failed to update prompt with id", idiomId)
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
		service.logger.Error(err, "Failed to create query with inputs")
		return nil, err
	}

	result, err := service.db.Exec(sql, args...)

	if err != nil {
		service.logger.Error(err, "Failed to create idiom inputs")
		return nil, err
	}

	affected, _ := result.RowsAffected()
	rows := int(affected)
	return &rows, nil
}

func (service *Service) CreateDescription(id string) (*models.IdiomDescription, error) {
	idioms := []models.Idiom{}
	idiomsQuery, args, err := sq.Select("*").From("idioms").Where("id = ?", id).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Error(err, "Failed to create a query", id)
		return nil, err
	}
	err = service.db.Select(&idioms, idiomsQuery, args...)
	if err != nil || len(idioms) == 0 {
		service.logger.Warn("Failed to query the idiom", id)
		return nil, err
	}
	idiom := idioms[0]
	textArgs := new(openai.TextCompletionArgs)
	textArgs.AddMessage("system", "You are the famous English teacher.")
	textArgs.AddMessage("system", "You are good at teaching English to countries in which people does not use English as a main language.")
	textArgs.AddMessage("system", "You have every knowledges to teach English to people.")
	textArgs.AddMessage("system", "Your missions are one task.")
	textArgs.AddMessage("system", "- Create a description explaining a situation with this idiom.")
	textArgs.AddMessage("system", "Your answer should be long and natural.")
	textArgs.AddMessage("system", "Your answer should be much more ORIGINAL content than others on the internet.")
	textArgs.AddMessage("system", "Your answer should be enough to use in real life.")
	textArgs.AddMessage("system", "Description should be longer than 300 letters.")
	textArgs.AddMessage("system", "Description should be shorted than 400 letters.")
	textArgs.AddMessage("system", "Description should not include abstract situations.")
	textArgs.AddMessage("system", "Description should include specific situations.")
	textArgs.AddMessage("system", "Response should be json format to {\"description\": string}")

	information := map[string]string{}
	information["idiom"] = idiom.Idiom
	information["meaning"] = idiom.MeaningBrief
	formatted, _ := json.Marshal(information)

	textArgs.AddMessage("assistant", fmt.Sprintf("The Idiom is here.\n%s\n", formatted))

	textArgs.AddMessage("user", "Create me a description suitable for explaining the situation with this idiom.")

	textArgs.Model = "gpt-4-turbo-preview"
	textArgs.Temperature = 0.8

	content, textError := service.ai.TextCompletion(textArgs)
	if textError != nil {
		service.logger.Error(textError, "Failed to create examples.", idiom.ID)
		return nil, textError
	}
	description := new(models.IdiomDescription)
	jsonError := json.Unmarshal([]byte(*content), description)
	if jsonError != nil {
		service.logger.Error(jsonError, "Failed to decode JSON.")
		return nil, jsonError
	}
	now := time.Now().UTC()
	publishedAt := now.Format(time.RFC3339Nano)
	updateQuery, args, err := sq.Update("idioms").Set("description", description.Description).Set("published_at", publishedAt).Where("id = ?", id).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Error(err, "Failed to update idiom", args...)
		return nil, err
	}
	_, err = service.db.Exec(updateQuery, args...)
	if err != nil {
		service.logger.Error(err, "Failed to update description with id", id)
		return nil, err
	}
	description.ID = id
	return description, nil
}

func (service *Service) CreateExamples(input models.IdiomInput) (*models.Idiom, error) {
	idioms := []models.Idiom{}

	idiomQuery, args, _ := sq.Select("*").From("idioms").Where("id = ?", input.ID).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	queryError := service.db.Select(&idioms, idiomQuery, args...)
	if queryError != nil {
		service.logger.Error(queryError, "Failed to query idioms with inputs")
		return nil, queryError
	}
	if len(idioms) > 0 {
		service.logger.Warn("Failed to query idioms with input")
		return nil, errors.New("failed to query idioms with input")
	}

	textArgs := new(openai.TextCompletionArgs)
	textArgs.AddMessage("system", "You are the famous English teacher")
	textArgs.AddMessage("system", "You are good at teaching English to countries in which people does not use English as a main language.")
	textArgs.AddMessage("system", "You have every knowledges to teach English to people.")
	textArgs.AddMessage("system", "Your missions are four tasks.")
	textArgs.AddMessage("system", "- Create a brief meaning")
	textArgs.AddMessage("system", "- Create a full meaning")
	textArgs.AddMessage("system", "- Create example sentences")
	textArgs.AddMessage("system", "- Create a description explaining a situation with this idiom.")
	textArgs.AddMessage("system", "Each your answer should be long and natural.")
	textArgs.AddMessage("system", "Your answer should be much more ORIGINAL content than others on the internet.")
	textArgs.AddMessage("system", "Your answer should be enough to use in real life.")
	textArgs.AddMessage("system", "The brief meaning should be longer than 120 letters.")
	textArgs.AddMessage("system", "The brief meaning should be shorter than 150 letters.")
	textArgs.AddMessage("system", "The full meaning should be longer than 1000 letters.")
	textArgs.AddMessage("system", "The full meaning should be shorter than 1100 letters.")
	textArgs.AddMessage("system", "You should create 10 example sentences.")
	textArgs.AddMessage("system", "Example sentences should be about 250 letters each sentence.")
	textArgs.AddMessage("system", "Example sentences should be more specific.")
	textArgs.AddMessage("system", "Example sentences should be less abstract.")
	textArgs.AddMessage("system", "Description should be longer than 300 letters.")
	textArgs.AddMessage("system", "Description should be shorted than 400 letters.")
	textArgs.AddMessage("system", "Description should not include abstract situations.")
	textArgs.AddMessage("system", "Description should include specific situations.")
	textArgs.AddMessage("system", "Response should be json format to {\"idiom\": string, \"meaningBrief\": string, \"meaningFull\": string, \"description\": string, \"examples\": [string]}")

	information := map[string]string{}
	information["idiom"] = input.Idiom
	information["meaning"] = input.Meaning
	formatted, _ := json.Marshal(information)

	textArgs.AddMessage("assistant", fmt.Sprintf("The Idiom is here.\n%s\n", formatted))

	textArgs.AddMessage("user", "Create me a brief meaning, a full meaning, a description and 10 example sentences.")

	textArgs.Model = "gpt-4-turbo-preview"
	textArgs.Temperature = 0.8

	content, textError := service.ai.TextCompletion(textArgs)
	if textError != nil {
		service.logger.Error(textError, "Failed to create examples with ", input.Idiom)
		return nil, errors.New("failed to create examples")
	}
	idiom := new(models.Idiom)

	jsonError := json.Unmarshal([]byte(*content), idiom)
	if jsonError != nil {
		service.logger.Error(jsonError, "Failed to decode JSON.")
		return nil, jsonError
	}
	idiomID := lib.ToIdiomID(idiom.Idiom)
	idiom.ID = idiomID

	if !idiom.Description.Valid || idiom.Examples == nil || len(idiom.Examples) == 0 {
		service.logger.Warn("Failed to create a description and examples by id", idiom.ID)
		return nil, errors.New("failed to create examples")
	}

	insertQuery, insertArgs, _ := sq.Insert("idioms").Columns("id", "idiom", "meaning_brief", "meaning_full", "description").Values(idiom.ID, idiom.Idiom, idiom.MeaningBrief, idiom.MeaningFull, idiom.Description).PlaceholderFormat(sq.Dollar).ToSql()
	_, insertError := service.db.Exec(insertQuery, insertArgs...)
	if insertError != nil {
		service.logger.Error(insertError, "Failed to insert idiom to database", idiom)

		return nil, insertError
	}

	exampleQuery := sq.Insert("idiom_examples").Columns("idiom_id", "expression")
	for _, example := range idiom.Examples {
		exampleQuery = exampleQuery.Values(idiom.ID, example)
	}
	exampleSql, exampleArgs, _ := exampleQuery.PlaceholderFormat(sq.Dollar).ToSql()
	_, exampleError := service.db.Exec(exampleSql, exampleArgs...)
	if exampleError != nil {
		service.logger.Error(exampleError, "Failed to insert idiom examples", idiom)

		return nil, exampleError
	}
	return idiom, nil
}
