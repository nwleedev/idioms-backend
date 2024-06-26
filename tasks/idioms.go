package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

func (task *Task) DeleteInput(input models.IdiomInput) {
	deleteQuery, deleteArgs, _ := sq.Delete("idiom_inputs").Where("id = ?", input.ID).PlaceholderFormat(sq.Dollar).ToSql()
	_, err := task.db.Exec(deleteQuery, deleteArgs...)

	if err != nil {
		task.logger.Error(err, "Failed to delete idiom input with id.", input.ID)
		return
	}
}

func (task *Task) CreateIdiomMeanings(interval time.Duration) {
	inputs := []models.IdiomInput{}
	idioms := []models.Idiom{}
	query, args, err := sq.Select("*").From("idiom_inputs").OrderBy("created_at asc").Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		task.logger.Error(err, "Failed to create a query from db.", args...)
		return
	}

	err = task.db.Select(&inputs, query, args...)
	if err != nil {
		task.logger.Error(err, "Failed to query a idiom input from db.")
		return
	}
	if len(inputs) == 0 {
		task.logger.Warn("There are no inputs.")
		return
	}
	input := inputs[0]

	idiomQuery, args, _ := sq.Select("*").From("idioms").Where("id = ?", input.ID).Limit(1).PlaceholderFormat(sq.Dollar).ToSql()
	err = task.db.Select(&idioms, idiomQuery, args...)
	if err != nil {
		task.logger.Error(err, "Failed to query idioms with inputs")
		return
	}
	if len(idioms) > 0 {
		task.logger.Warn("The idiom already exists", input)
		task.DeleteInput(input)
		return
	}

	textArgs := new(openai.TextCompletionArgs)
	textArgs.AddMessage("system", "You are the well telanted English instructor.")
	textArgs.AddMessage("system", "You are good at teaching English to everyone.")
	textArgs.AddMessage("system", "You should know how to teach English to students.")
	textArgs.AddMessage("system", "You have every knowledges to teach English to people.")
	textArgs.AddMessage("system", "You should have the mindset to make English learning textbooks for high school students.")
	textArgs.AddMessage("system", "You should act like that you are writing educational books for high school students.")
	textArgs.AddMessage("system", "Your answer should be enough to use in real life.")
	textArgs.AddMessage("system", "You should use active tones instead of passive tones.")
	textArgs.AddMessage("system", "You should use active voices instead of passive voices.")
	textArgs.AddMessage("system", "Your content should be much more unique than plagiarism content.")
	textArgs.AddMessage("system", "Your content should be academic.")
	textArgs.AddMessage("system", "Your content should be extremely detailed.")
	textArgs.AddMessage("system", "Your content should be highly readable.")
	textArgs.AddMessage("system", "I will be very disappointed if your answer is like plagiarism.")

	textArgs.AddMessage("system", "Your missions are tasks below.")
	textArgs.AddMessage("system", "- Create a brief meaning.")
	textArgs.AddMessage("system", "- Create a full meaning.")
	textArgs.AddMessage("system", "- Create example sentences.")
	textArgs.AddMessage("system", "You should limit the brief meaning to 200 words.")
	textArgs.AddMessage("system", "You should limit the full meaning to 1000 words.")
	textArgs.AddMessage("system", "You should create 10 example sentences.")
	textArgs.AddMessage("system", "- You should limit each example to 500 words.")
	textArgs.AddMessage("system", "- Each example should be like examples in Harvard dictionary.")
	textArgs.AddMessage("system", "- Each example can be academic, casual, or businesslike.")
	textArgs.AddMessage("system", "- Each example should be shorter than 600 letters.")
	textArgs.AddMessage("system", "- Each example should be much detailed than plagiarism content.")
	textArgs.AddMessage("system", "- Each example should be more practical and specific to use in real life.")
	textArgs.AddMessage("system", "Response should be json format to {\"idiom\": \"A Idiom\", \"meaningBrief\": \"This is a brief meaning\", \"meaningFull\": \"This is a full meaning.\", \"examples\": [\"This is example 1.\", \"This is example 2.\"]}")
	information := map[string]string{}
	information["idiom"] = input.Idiom
	information["meaning"] = input.Meaning
	formatted, _ := json.Marshal(information)

	textArgs.AddMessage("assistant", fmt.Sprintf("The Idiom is here.\n%s\n", string(formatted)))

	textArgs.AddMessage("user", fmt.Sprintf("Create me a brief meaning, a full meaning, a description and 10 example sentences with this idiom %s.", string(formatted)))

	textArgs.Model = "gpt-4o"
	textArgs.Temperature = 1

	content, err := task.ai.TextCompletion(textArgs)
	if err != nil {
		task.logger.Error(err, "Failed to create examples.", input.Idiom)
		return
	}
	idiom := new(models.Idiom)

	trimmed := strings.TrimLeft(*content, "```json")
	trimmed = strings.TrimRight(trimmed, "```")

	err = json.Unmarshal([]byte(trimmed), idiom)
	if err != nil {
		task.logger.Error(err, "Failed to decode JSON.", content)

		now := time.Now().Unix()
		fileLog, _ := os.Create(fmt.Sprintf("./logs/%d.json", now))
		logContent, _ := json.Marshal(*content)
		fileLog.Write(logContent)
		return
	}
	idiomID := lib.ToIdiomID(idiom.Idiom)
	idiom.ID = idiomID

	if !idiom.Description.Valid || idiom.Examples == nil || len(idiom.Examples) == 0 {
		task.logger.Warn("Failed to create examples.", idiom)
		return
	}

	insertQuery, insertArgs, _ := sq.Insert("idioms").Columns("id", "idiom", "meaning_brief", "meaning_full", "description").Values(idiom.ID, idiom.Idiom, idiom.MeaningBrief, idiom.MeaningFull, idiom.Description).PlaceholderFormat(sq.Dollar).ToSql()
	_, err = task.db.Exec(insertQuery, insertArgs...)
	if err != nil {
		task.logger.Error(err, "Failed to insert idiom.", idiom)

		task.DeleteInput(input)
		return
	}
	exampleQuery := sq.Insert("idiom_examples").Columns("idiom_id", "expression")
	for _, example := range idiom.Examples {
		exampleQuery = exampleQuery.Values(idiom.ID, example)
	}
	exampleSql, exampleArgs, _ := exampleQuery.PlaceholderFormat(sq.Dollar).ToSql()
	_, err = task.db.Exec(exampleSql, exampleArgs...)
	if err != nil {
		task.logger.Error(err, "Failed to insert examples.", idiom)

		task.DeleteInput(input)
		return
	}
	task.DeleteInput(input)
}
