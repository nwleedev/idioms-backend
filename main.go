package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	"github.com/nw.lee/idioms-backend/handler"
	"github.com/nw.lee/idioms-backend/idioms"
	"github.com/nw.lee/idioms-backend/logger"
	"github.com/nw.lee/idioms-backend/openai"
	"github.com/nw.lee/idioms-backend/storage"
	"github.com/nw.lee/idioms-backend/tasks"
	"github.com/nw.lee/idioms-backend/thumbnail"
)

func main() {
	godotenv.Load("./.aws")
	godotenv.Load("./.env")

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbSslmode := os.Getenv("DB_SSLMODE")
	dbSource := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, dbSslmode)

	conn, err := sqlx.Connect("pgx", dbSource)
	if err != nil {
		panic(err)
	}
	awsId := os.Getenv("aws_access_key_id")
	awsKey := os.Getenv("aws_secret_access_key")
	awsRegion := os.Getenv("region")
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		panic(err)
	}
	loggerService := logger.NewService(log.Default())

	aiKey := os.Getenv("OPENAI_API_KEY")
	orgId := os.Getenv("OPENAI_ORG")
	awsRoleArn := os.Getenv("AWS_ROLE_ARN")

	var isAdmin bool

	if os.Getenv("IS_ADMIN") == "true" {
		isAdmin = true
	} else {
		isAdmin = false
	}
	aiService := openai.NewOpenAi(aiKey, orgId, loggerService)
	storageService := storage.NewService(&awsConfig, awsId, awsKey, awsRoleArn)

	idiomService := idioms.NewService(conn, loggerService)

	thumbnailContext := context.Background()

	thumbnailService := thumbnail.NewService(conn, loggerService, storageService, aiService, &thumbnailContext)
	idiomController := idioms.NewController(idiomService, thumbnailService, loggerService)

	handler := handler.NewHandler(isAdmin).AddIdiomController(idiomController)

	if isAdmin {
		idiomTask := tasks.NewIdiomTask(conn, loggerService, aiService)

		go func() {
			for {
				idiomTask.CreateIdiomMeanings(time.Second * time.Duration(4))
				time.Sleep(time.Minute * time.Duration(2))
			}
		}()
	}

	handler.Run()
	handler.ListenAndServe(":8081")
}
