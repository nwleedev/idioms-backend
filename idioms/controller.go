package idioms

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nw.lee/idioms-backend/lib"
	"github.com/nw.lee/idioms-backend/logger"
	"github.com/nw.lee/idioms-backend/models"
	"github.com/nw.lee/idioms-backend/thumbnail"
)

type Controller struct {
	idiomService     IdiomService
	thumbnailService thumbnail.ThumbnailService

	logger logger.LoggerService
}

type IdiomController interface {
	GetIdiomById(writer http.ResponseWriter, request *http.Request)
	GetRelatedIdioms(writer http.ResponseWriter, request *http.Request)
	GetIdioms(writer http.ResponseWriter, request *http.Request)
	GetMainPageIdioms(writer http.ResponseWriter, request *http.Request)
	SearchIdioms(writer http.ResponseWriter, request *http.Request)
	GetIdiomsWithThumbnail(writer http.ResponseWriter, request *http.Request)
	UploadThumbnail(writer http.ResponseWriter, request *http.Request)
	CreateThumbnail(writer http.ResponseWriter, request *http.Request)
	CreateThumbnailByURL(writer http.ResponseWriter, request *http.Request)
	UpdateThumbnailPrompt(writer http.ResponseWriter, request *http.Request)
	CreateIdiomInputs(writer http.ResponseWriter, request *http.Request)
	CreateDescription(writer http.ResponseWriter, request *http.Request)
}

func NewController(idiomService IdiomService, thumbnailService thumbnail.ThumbnailService, logger logger.LoggerService) *Controller {
	controller := new(Controller)
	controller.idiomService = idiomService
	controller.thumbnailService = thumbnailService
	controller.logger = logger

	return controller
}

func (contoller *Controller) EncodeToken(idioms []models.Idiom, filter *QueryFilter) (*models.CursorToken, error) {
	if len(idioms) < 1 {
		return nil, errors.New("failed to query idioms")
	}
	fromIdiom := idioms[0]
	toIdiom := idioms[len(idioms)-1]

	prevCursor := new(Cursor)
	nextCursor := new(Cursor)
	if filter.OrderBy == "idiom" {
		prevCursor.Idiom = &(fromIdiom.Idiom)
		nextCursor.Idiom = &(toIdiom.Idiom)
	} else {
		prevCursor.CreatedAt = &(fromIdiom.CreatedAt)
		nextCursor.CreatedAt = &(toIdiom.CreatedAt)
	}
	nextCursor.IsNext = true
	prevCursor.IsNext = false

	prevToken, prevError := json.Marshal(prevCursor)
	nextToken, nextError := json.Marshal(nextCursor)
	if prevError != nil || nextError != nil {
		contoller.logger.Error(errors.New("failed to create cursors"), "2 Errors", prevError, nextError)
		return nil, errors.New("failed to create cursors")
	}
	encodedPrevToken := base64.StdEncoding.EncodeToString([]byte(prevToken))
	encodedNextToken := base64.StdEncoding.EncodeToString([]byte(nextToken))
	return &models.CursorToken{
		Previous: encodedPrevToken,
		Next:     encodedNextToken,
	}, nil
}

func (controller *Controller) DecodeToken(request *http.Request) *Cursor {
	var encodedToken string
	params := request.URL.Query()
	encodedNextToken := params.Get("nextToken")
	encodedPrevToken := params.Get("prevToken")
	cursor := new(Cursor)

	if len(encodedPrevToken) == 0 && len(encodedNextToken) == 0 {
		return nil
	}

	if len(encodedPrevToken) > 0 {
		encodedToken = encodedPrevToken
	} else {
		encodedToken = encodedNextToken
	}
	token, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		controller.logger.Error(err, "failed to decode tokens.", encodedToken)
		return nil
	}
	err = json.Unmarshal(token, cursor)
	if err != nil {
		controller.logger.Error(err, "failed to decode JSON.", encodedToken)
		return nil
	}
	return cursor
}

func (controller *Controller) GetFilter(request *http.Request) (*QueryFilter, error) {
	params := request.URL.Query()
	filter := new(QueryFilter)
	orderBy := (params.Get("orderBy"))
	orderDirection := strings.ToLower(params.Get(("orderDirection")))
	count, intErr := strconv.Atoi(params.Get(("count")))
	if intErr != nil {
		count = 20
	}
	filter.Count = count
	cursor := controller.DecodeToken(request)
	operator := "<"
	innerOrderDirection := "desc"

	switch orderBy {
	case "createdAt":
		{
			filter.OrderBy = "created_at"
			break
		}
	case "idiom":
		{
			filter.OrderBy = "idiom"
			break
		}
	default:
		{
			filter.OrderBy = "published_at"
		}
	}

	if orderDirection != "asc" {
		orderDirection = "desc"
	}
	filter.OrderDirection = orderDirection
	if cursor != nil {
		filter.idiom = cursor.Idiom
		filter.createdAt = cursor.CreatedAt
		if filter.OrderDirection == "desc" && cursor.IsNext {
			innerOrderDirection = "desc"
			operator = "<"
		}
		if filter.OrderDirection == "desc" && !cursor.IsNext {
			innerOrderDirection = "asc"
			operator = ">"
		}
		if filter.OrderDirection == "asc" && cursor.IsNext {
			innerOrderDirection = "asc"
			operator = ">"
		}
		if filter.OrderDirection == "asc" && !cursor.IsNext {
			innerOrderDirection = "desc"
			operator = "<"
		}
	}
	filter.innerOrderDirection = innerOrderDirection
	filter.operator = operator

	return filter, nil
}

func (controller *Controller) GetIdiomById(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("content-type", "application/json")
	id := chi.URLParam(request, "id")
	idiom, err := controller.idiomService.GetIdiomById(id)
	body := map[string]interface{}{
		"idiom": nil,
	}

	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	body["idiom"] = idiom
	str, _ := json.Marshal(body)
	writer.Write(str)
}

func (controller *Controller) GetRelatedIdioms(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("content-type", "application/json")
	idiomId := chi.URLParam(request, "id")
	body := map[string]interface{}{
		"idioms": nil,
	}
	idioms, err := controller.idiomService.GetRelatedIdioms(idiomId)
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	body["idioms"] = idioms
	str, _ := json.Marshal(body)
	writer.Write(str)
}

func (controller *Controller) GetIdioms(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("content-type", "application/json")
	filter, err := controller.GetFilter(request)
	body := new(models.IdiomResponse)
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	idioms, err := controller.idiomService.GetIdioms(filter, false)
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}

	cursorToken, err := controller.EncodeToken(idioms, filter)
	if err != nil {
		controller.logger.Warn("failed to create cursor tokens.", err)
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	body.Cursor.Previous = cursorToken.Previous
	body.Cursor.Next = cursorToken.Next
	body.Idioms = idioms
	str, _ := json.Marshal(body)
	writer.Write(str)
}

func (controller *Controller) SearchIdioms(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("content-type", "application/json")
	filter, err := controller.GetFilter(request)
	searchQuery := request.URL.Query()
	filter.Keyword = searchQuery.Get("keyword")
	body := new(models.IdiomResponse)
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	idioms, err := controller.idiomService.SearchIdioms(filter, true)
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}

	cursorToken, err := controller.EncodeToken(idioms, filter)
	if err != nil {
		controller.logger.Warn("failed to create cursor tokens.", err)
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	body.Cursor.Previous = cursorToken.Previous
	body.Cursor.Next = cursorToken.Next
	body.Idioms = idioms
	str, _ := json.Marshal(body)
	writer.Write(str)
}

func (controller *Controller) GetIdiomsWithThumbnail(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("content-type", "application/json")
	filter, err := controller.GetFilter(request)
	body := new(models.IdiomResponse)
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	idioms, err := controller.idiomService.GetIdioms(filter, true)
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}

	cursorToken, err := controller.EncodeToken(idioms, filter)
	if err != nil {
		controller.logger.Warn("failed to create cursor tokens.", err)
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}
	body.Cursor.Previous = cursorToken.Previous
	body.Cursor.Next = cursorToken.Next
	body.Idioms = idioms
	str, _ := json.Marshal(body)
	writer.Write(str)
}

func (controller *Controller) GetMainPageIdioms(writer http.ResponseWriter, request *http.Request) {
	body := map[string]interface{}{
		"idioms": nil,
	}
	writer.Header().Add("content-type", "application/json")

	idioms, err := controller.idiomService.GetMainPageIdioms()
	if err != nil {
		str, _ := json.Marshal(body)
		writer.Write(str)
		return
	}

	body["idioms"] = idioms
	str, _ := json.Marshal(body)
	writer.Write(str)
}

func (controller *Controller) UploadThumbnail(writer http.ResponseWriter, request *http.Request) {
	message := map[string]interface{}{
		"idiomId":   nil,
		"thumbnail": nil,
	}

	formSize := 32 << 20
	formBufferSize := 4 << 20
	_ = formBufferSize
	// request.Body = http.MaxBytesReader(writer, request.Body, int64(formSize)+int64(formBufferSize))
	err := request.ParseMultipartForm(int64(formSize))

	writer.Header().Add("content-type", "application/json")

	if err != nil {
		controller.logger.Error(err, "Failed to parse form.")
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}

	idiomId := request.FormValue("idiomId")

	message = map[string]interface{}{
		"idiomId":   idiomId,
		"thumbnail": nil,
	}

	formFile, handler, err := request.FormFile("thumbnail")
	if err != nil {
		controller.logger.Error(err, "Failed to get file from form", idiomId)
		message["idiomId"] = idiomId
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	file := &lib.File{
		Content:   formFile,
		Extension: path.Ext(handler.Filename),
	}

	thumbnail, err := controller.thumbnailService.UploadThumbnail(idiomId, file)
	if err != nil {
		message["idiomId"] = idiomId
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}

	message["thumbnail"] = thumbnail
	str, _ := json.Marshal(message)
	writer.Write(str)
}

func (controller *Controller) CreateDescription(writer http.ResponseWriter, request *http.Request) {
	message := map[string]interface{}{
		"id":          nil,
		"description": nil,
	}
	idiomId := chi.URLParam(request, "id")
	description, err := controller.idiomService.CreateDescription(idiomId)
	if err != nil {
		message["idiomId"] = idiomId
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	message["id"] = idiomId
	message["description"] = description.Description
	str, _ := json.Marshal(message)
	writer.Write(str)
}

func (controller *Controller) CreateThumbnailByURL(writer http.ResponseWriter, request *http.Request) {
	message := map[string]interface{}{
		"idiomId":   nil,
		"thumbnail": nil,
	}
	idiomId := request.FormValue("idiomId")
	imageUrl := request.FormValue("imageUrl")
	message["idiomId"] = idiomId
	decodedUrl, _ := base64.StdEncoding.DecodeString(imageUrl)

	thumbnail, err := controller.thumbnailService.CreateThumbnailByURL(idiomId, string(decodedUrl))
	if err != nil {
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	message["thumbnail"] = thumbnail
	str, _ := json.Marshal(message)
	writer.Write(str)
}

func (controller *Controller) UpdateThumbnailPrompt(writer http.ResponseWriter, request *http.Request) {
	message := map[string]interface{}{
		"idiomId":         nil,
		"thumbnailPrompt": nil,
	}
	id := chi.URLParam(request, "id")
	body := new(models.IdiomThumbnailBody)
	err := json.NewDecoder(request.Body).Decode(body)
	if err != nil {
		controller.logger.Error(err, "Failed to decode JSON.", id)
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	_, err = controller.idiomService.UpdateThumbnailPrompt(id, body.ThumbnailPrompt)
	if err != nil {
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	message["thumbnailPrompt"] = body.ThumbnailPrompt
	str, _ := json.Marshal(message)
	writer.Write(str)
}

func (controller *Controller) CreateIdiomInputs(writer http.ResponseWriter, request *http.Request) {
	message := map[string]interface{}{
		"rows": nil,
	}
	inputs := []models.IdiomInput{}
	err := json.NewDecoder(request.Body).Decode(&inputs)
	if err != nil {
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}

	rows, err := controller.idiomService.CreateIdiomInputs(inputs)
	if err != nil {
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	message["rows"] = &rows
	str, _ := json.Marshal(message)
	writer.Write(str)
}

func (controller *Controller) CreateThumbnail(writer http.ResponseWriter, request *http.Request) {
	message := map[string]interface{}{
		"image": nil,
	}
	input := new(models.IdiomImageInput)
	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	image, err := controller.thumbnailService.CreateThumbnail(input.Prompt)
	if err != nil {
		str, _ := json.Marshal(message)
		writer.Write(str)
		return
	}
	message["image"] = *image
	str, _ := json.Marshal(message)
	writer.Write(str)
}
