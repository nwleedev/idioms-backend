package tasks

import (
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgtype"
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
	textArgs := new(openai.TextCompletionArgs)
	textArgs.AddMessage("system", "You are the famous English teacher, You are good at teaching English to countries in which people does Korean, Japanese instead of English.")
	textArgs.AddMessage("system", "Your answer should be long and natural as soon as we can use these expressions in real world. Brief meaning should satisfy from 100 to 120 characters. Full meaning should be long as you as possible. Each expression should be longer than 400 characters.")
	textArgs.AddMessage("system", fmt.Sprintf("The idiom is %s. The meaning of this idiom is %s. Make ten expressions with this please.", input.Idiom, input.Meaning))
	textArgs.AddMessage("system", "Each expressions should be longer than 500 characters.")
	textArgs.AddMessage("system", "Response should be json format to {\"idiom\": string, \"meaningBrief\": string, \"meaningFull\": string, \"examples\": [string]}")

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

	promptArgs := new(openai.TextCompletionArgs)
	promptArgs.Model = "gpt-4"
	promptArgs.Temperature = 0.8

	promptArgs.AddMessage("system", "You are a image generation prompt engineer. You are telanted for writing prompt of image generation ai.")
	promptArgs.AddMessage("system", "You have knowledges of how to write prompt for multi image generation with ai services, such as Dall-E, Bing Ai, Stable diffusion.")
	promptArgs.AddMessage("system", "Your mission is building prompts for me. I want to make thumbnails for vocabulary applications. Each card in this app should show thumbnail related to following idioms.")
	promptArgs.AddMessage("system", "You should give me the prompt of Stable diffusion. Prompt should be 70 ~ 80 characters. Prompt should not include letters or alphabets.")
	promptArgs.AddMessage("system", "Each thumbnails should show their correct meanings")
	promptArgs.AddMessage("system", "The prompt shouldn't be abstract.")
	promptArgs.AddMessage("system", "Response format should be {\"idiom\": string, \"prompt\": string}")
	promptArgs.AddMessage("system", fmt.Sprintf("Idiom is %s, Meaning is %s.", input.Idiom, input.Meaning))

	promptContent, err := task.ai.TextCompletion(promptArgs)
	if err != nil {
		task.logger.Println("Failed to query a thumbnail prompt from db with id %s", idiom.ID)
		task.logger.PrintError("", err)
		return
	}
	prompt := new(models.IdiomPrompt)
	err = json.Unmarshal([]byte(*promptContent), prompt)
	if err != nil {
		task.logger.Println("Failed to decode prompt response", promptContent)
		task.logger.PrintError("", err)
		return
	}

	idiom.ThumbnailPrompt = pgtype.Text{
		Valid:  true,
		String: prompt.Prompt,
	}

	if !idiom.ThumbnailPrompt.Valid || idiom.Examples == nil || len(idiom.Examples) == 0 {
		task.logger.Println("Failed to create a thumbnail prompt and examples by id %s", idiom.ID)
		return
	}

	insertQuery, insertArgs, err := sq.Insert("idioms").Columns("id", "idiom", "meaning_brief", "meaning_full", "thumbnail_prompt").Values(idiom.ID, idiom.Idiom, idiom.MeaningBrief, idiom.MeaningFull, idiom.ThumbnailPrompt).PlaceholderFormat(sq.Dollar).ToSql()
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

	deleteQuery, deleteArgs, _ := sq.Delete("idiom_inputs").Where("id = ?", idiom.ID).PlaceholderFormat(sq.Dollar).ToSql()
	_, err = task.db.Exec(deleteQuery, deleteArgs...)

	if err != nil {
		task.logger.Println("Failed to delete idiom input with idiom id %s", idiom.ID)
		task.logger.PrintError("", err)
		return
	}
}
