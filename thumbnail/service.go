package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jmoiron/sqlx"
	"github.com/nw.lee/idioms-backend/lib"
	"github.com/nw.lee/idioms-backend/logger"
	"github.com/nw.lee/idioms-backend/openai"
	"github.com/nw.lee/idioms-backend/storage"
)

type ThumbnailService interface {
	UploadThumbnail(idiomId string, file *lib.File) (*string, error)
	CreateThumbnailByURL(idiomId string, url string) (*string, error)
	CreateThumbnail(prompt string) (*string, error)
}

type Service struct {
	db      *sqlx.DB
	logger  logger.LoggerService
	storage storage.StorageService
	ai      openai.OpenAiInterface
	context *context.Context
}

var bucketName string = "austin-idioms"

func NewService(db *sqlx.DB, logger logger.LoggerService, storage storage.StorageService, ai openai.OpenAiInterface, context *context.Context) *Service {
	service := new(Service)
	service.db = db
	service.logger = logger
	service.storage = storage
	service.context = context
	service.ai = ai

	return service
}

func (service *Service) CreateThumbnailByURL(idiomId string, url string) (*string, error) {
	resp, err := http.Get(url)
	if err != nil {
		service.logger.Println("Failed to fetch image with url %s.", url)
		service.logger.PrintError("Error: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	now := time.Now().UTC()
	contentType := resp.Header.Get("content-type")
	extension := strings.Split(contentType, "/")[1]
	fileKey := fmt.Sprintf("%d/%d/%d/%s.%s", now.Year(), now.Month(), now.Day(), idiomId, extension)
	imageBytes := new(bytes.Buffer)
	io.Copy(imageBytes, resp.Body)

	output, err := service.storage.GetStorage().PutObject(*service.context, &s3.PutObjectInput{
		Bucket:      &bucketName,
		Key:         &fileKey,
		Body:        bytes.NewReader(imageBytes.Bytes()),
		ContentType: &contentType,
	})

	if err != nil || output == nil {
		service.logger.Println("Failed to create a thumbnail with id %s.", idiomId)
		service.logger.PrintError("Error: %s", err)
		return nil, err
	}
	query, args, err := sq.Update("idioms").Set("thumbnail", fileKey).Where("id = ?", idiomId).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Println("Failed to query the idiom with id %s.", idiomId)
		return nil, err
	}
	_, err = service.db.Exec(query, args...)
	if err != nil {
		service.logger.Println("Failed to update the idiom with id %s.", idiomId)
		service.logger.PrintError("Error: %s", err)
		return nil, err
	}

	return &fileKey, nil
}

func (service *Service) UploadThumbnail(idiomId string, file *lib.File) (*string, error) {
	now := time.Now().UTC()
	fileKey := fmt.Sprintf("%d/%d/%d/%s%s", now.Year(), now.Month(), now.Day(), idiomId, file.Extension)
	contentType := fmt.Sprintf("image/%s", strings.ReplaceAll(file.Extension, ".", ""))

	output, err := service.storage.GetStorage().PutObject(*service.context, &s3.PutObjectInput{
		Bucket:      &bucketName,
		Key:         &fileKey,
		Body:        file.Content,
		ContentType: &contentType,
	})

	if err != nil || output == nil {
		service.logger.Println("Failed to create a thumbnail with id %s.", idiomId)
		service.logger.PrintError("Error: %s", err)
		return nil, err
	}
	query, args, err := sq.Update("idioms").Set("thumbnail", fileKey).Where("id = ?", idiomId).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		service.logger.Println("Failed to query the idiom with id %s.", idiomId)
		return nil, err
	}
	_, err = service.db.Exec(query, args...)
	if err != nil {
		service.logger.Println("Failed to update the idiom with id %s.", idiomId)
		service.logger.PrintError("Error: %s", err)
		return nil, err
	}

	return &fileKey, nil
}

func (service *Service) CreateThumbnail(prompt string) (*string, error) {
	image, err := service.ai.Image(prompt)
	if err != nil {
		service.logger.Println("Failed to create thumbnail with prompt %s", prompt)
		return nil, err
	}

	resp, err := http.Get(*image)
	if err != nil {
		service.logger.Println("Failed to fetch image with url %s.", *image)
		service.logger.PrintError("Error: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("content-type")
	extension := strings.Split(contentType, "/")[1]
	fileKey := fmt.Sprintf("drafts/output.%s", extension)
	imageBytes := new(bytes.Buffer)
	io.Copy(imageBytes, resp.Body)

	output, err := service.storage.GetStorage().PutObject(*service.context, &s3.PutObjectInput{
		Bucket:      &bucketName,
		Key:         &fileKey,
		Body:        bytes.NewReader(imageBytes.Bytes()),
		ContentType: &contentType,
	})

	if err != nil || output == nil {
		service.logger.Println("Failed to save a draft image.")
		service.logger.PrintError("Error: %s", err)
		return nil, err
	}

	return &fileKey, nil
}
