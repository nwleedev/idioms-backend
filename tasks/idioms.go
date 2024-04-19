package tasks

import (
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/nw.lee/idioms-backend/lib"
	"github.com/nw.lee/idioms-backend/logger"
	"github.com/nw.lee/idioms-backend/models"
	"github.com/nw.lee/idioms-backend/openai"
)

type IdiomTask interface {
}

type Task struct {
	db     *sqlx.DB
	logger logger.LoggerService
	ai     openai.OpenAiInterface
}

func NewIdiomTask(db *sqlx.DB, logger logger.LoggerService, ai openai.OpenAiInterface) *Task {
	task := new(Task)
	task.db = db
	task.logger = logger
	task.ai = ai

	return task
}

func (task *Task) CreateIdiomMeanings(interval time.Duration) {
	inputs := []models.IdiomInput{}
	idioms := []models.Idiom{}
	query, args, err := sq.Select("*").From("idiom_inputs").OrderBy("created_at asc").Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	_ = args
	if err != nil {
		task.logger.Println("Failed to generate a query from db")
		task.logger.PrintError("", err)
		return
	}

	err = task.db.Select(&inputs, query, args...)
	if err != nil {
		task.logger.Println("Failed to query a idiom input from db")
		task.logger.PrintError("", err)
		return
	}
	if len(inputs) == 0 {
		task.logger.Println("No inputs")
		return
	}
	input := inputs[0]

	deleteQuery, deleteArgs, _ := sq.Delete("idiom_inputs").Where("id = ?", input.ID).PlaceholderFormat(sq.Dollar).ToSql()
	_, err = task.db.Exec(deleteQuery, deleteArgs...)

	if err != nil {
		task.logger.Println("Failed to delete idiom input with idiom id %s", input.ID)
		task.logger.PrintError("", err)
		return
	}

	idiomQuery, args, _ := sq.Select("*").From("idioms").Where("id = ?", input.ID).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	err = task.db.Select(&idioms, idiomQuery, args...)
	if err != nil {
		task.logger.PrintError("Failed to query idioms with inputs", err)
		return
	}
	if len(idioms) > 0 {
		task.logger.Println("Idiom exists with id %s", input.ID)
		deleteQuery, deleteArgs, _ := sq.Delete("idiom_inputs").Where("id = ?", input.ID).PlaceholderFormat(sq.Dollar).ToSql()
		_, err = task.db.Exec(deleteQuery, deleteArgs...)

		if err != nil {
			task.logger.Println("Failed to delete idiom input with idiom id %s", input.ID)
			task.logger.PrintError("", err)
			return
		}
		return
	}
	textArgs := new(openai.TextCompletionArgs)
	textArgs.AddMessage("system", "You are the famous English teacher")
	textArgs.AddMessage("system", "You are good at teaching English to countries in which people does not use English as a main language.")
	textArgs.AddMessage("system", "You have every knowledges to teach English to people.")
	textArgs.AddMessage("system", "Your missions are three tasks.")
	textArgs.AddMessage("system", "- Create a brief meaning")
	textArgs.AddMessage("system", "- Create a full meaning")
	textArgs.AddMessage("system", "- Create example sentences")
	textArgs.AddMessage("system", "- Create a description explaining a situation with this idiom.")
	textArgs.AddMessage("system", "Each your answers should be long and natural.")
	textArgs.AddMessage("system", "Your answer should be much more ORIGINAL content than others on the internet.")
	textArgs.AddMessage("system", "Your answer should be enough to use in real life.")
	textArgs.AddMessage("system", "Brief meaning should satisfy from 120 to 140 letters.")
	textArgs.AddMessage("system", "Full meaning should satisfy about 1000 letters.")
	textArgs.AddMessage("system", "Each examples should be longer than 500 letters.")
	textArgs.AddMessage("system", "Description should be about 500 letters.")
	textArgs.AddMessage("system", "Description should not include abstract situations.")
	textArgs.AddMessage("system", "Description should include specific situations.")
	textArgs.AddMessage("system", "Response should be json format to {\"idiom\": string, \"meaningBrief\": string, \"meaningFull\": string, \"description\": string, \"examples\": [string]}")

	textArgs.AddMessage("assistant", fmt.Sprintf("The idiom is %s. The meaning of this idiom is %s.", input.Idiom, input.Meaning))

	textArgs.AddMessage("user", "Create me a brief meaning, a full meaning, a description and example sentences with following data.")

	textArgs.Model = "gpt-4"
	textArgs.Temperature = 0.8

	content, err := task.ai.TextCompletion(textArgs)
	if err != nil {
		task.logger.Println("Failed to make examples from %s", input.Idiom)
		task.logger.PrintError("", err)
		return
	}
	idiom := new(models.Idiom)

	err = json.Unmarshal([]byte(*content), idiom)
	if err != nil {
		task.logger.Println("Failed to decode json", content)
		task.logger.PrintError("", err)
		return
	}
	idiomID := lib.ToIdiomID(idiom.Idiom)
	idiom.ID = idiomID

	if !idiom.Description.Valid || idiom.Examples == nil || len(idiom.Examples) == 0 {
		task.logger.Println("Failed to create a thumbnail prompt and examples by id %s", idiom.ID)
		return
	}

	insertQuery, insertArgs, err := sq.Insert("idioms").Columns("id", "idiom", "meaning_brief", "meaning_full", "description").Values(idiom.ID, idiom.Idiom, idiom.MeaningBrief, idiom.MeaningFull, idiom.Description).PlaceholderFormat(sq.Dollar).ToSql()
	_, err = task.db.Exec(insertQuery, insertArgs...)
	if err != nil {
		task.logger.Println("Failed to insert idiom with id %s", idiom.ID)
		task.logger.PrintError("", err)
		return
	}
	exampleQuery := sq.Insert("idiom_examples").Columns("idiom_id", "expression")
	for _, example := range idiom.Examples {
		exampleQuery = exampleQuery.Values(idiom.ID, example)
	}
	exampleSql, exampleArgs, _ := exampleQuery.PlaceholderFormat(sq.Dollar).ToSql()
	_, err = task.db.Exec(exampleSql, exampleArgs...)
	if err != nil {
		task.logger.Println("Failed to insert idiom example with idiom id %s", idiom.ID)
		task.logger.PrintError("", err)
		return
	}
}
