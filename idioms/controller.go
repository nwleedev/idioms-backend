package idioms

import (
	"encoding/base64"
	"encoding/json"
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
	SearchIdioms(writer http.ResponseWriter, request *http.Request)
	GetIdiomsWithThumbnail(writer http.ResponseWriter, request *http.Request)
	UploadThumbnail(writer http.ResponseWriter, request *http.Request)
	CreateThumbnail(writer http.ResponseWriter, request *http.Request)
	CreateThumbnailByURL(writer http.ResponseWriter, request *http.Request)
	UpdateThumbnailPrompt(writer http.ResponseWriter, request *http.Request)
	CreateIdiomInputs(writer http.ResponseWriter, request *http.Request)
}

func NewController(idiomService IdiomService, thumbnailService thumbnail.ThumbnailService, logger logger.LoggerService) *Controller {
	controller := new(Controller)
	controller.idiomService = idiomService
	controller.thumbnailService = thumbnailService
	controller.logger = logger

	return controller
}

func (contoller *Controller) EncodeToken(idioms []models.Idiom, filter *QueryFilter, hasPrevous bool) (string, error) {
	if len(idioms) < 1 {
		return "", nil
	}
	var idiom models.Idiom
	if hasPrevous {
		idiom = idioms[0]
	} else {
		idiom = idioms[len(idioms)-1]
	}

	cursor := new(Cursor)
	if filter.OrderBy == "idiom" {
		cursor.Idiom = &(idiom.Idiom)
	} else {
		cursor.CreatedAt = &(idiom.CreatedAt)
	}

	nextToken, err := json.Marshal(cursor)
	encodedNextToken := base64.StdEncoding.EncodeToString([]byte(nextToken))
	return encodedNextToken, err
}

func (controller *Controller) DecodeToken(request *http.Request, hasPrevious bool) *Cursor {
	var encodedToken string
	var tokenName string = "nextToken"
	if hasPrevious {
		tokenName = "previousToken"
	}

	params := request.URL.Query()
	encodedToken = params.Get(tokenName)
	cursor := new(Cursor)
	token, _ := base64.StdEncoding.DecodeString(encodedToken)
	err := json.Unmarshal(token, cursor)
	if err != nil {
		return nil
	}
	return cursor
}

func (controller *Controller) GetFilter(request *http.Request) (*QueryFilter, error) {
	params := request.URL.Query()
	filter := new(QueryFilter)
	orderBy := strings.ToLower(params.Get("orderBy"))
	orderDirection := strings.ToLower(params.Get(("orderDirection")))
	count, intErr := strconv.Atoi(params.Get(("count")))
	if intErr != nil {
		count = 10
	}
	operator := "<"
	cursor := new(Cursor)
	previousCursor := controller.DecodeToken(request, true)
	nextCursor := controller.DecodeToken(request, false)
	hasPrevious := false
	if nextCursor != nil {
		cursor = nextCursor
	}
	if previousCursor != nil {
		cursor = previousCursor
		hasPrevious = true
	}

	if orderBy != "idiom" && orderBy != "created_at" {
		orderBy = "created_at"
	}
	if orderDirection != "desc" && orderDirection != "asc" {
		if hasPrevious {
			orderDirection = "asc"
		} else {
			orderDirection = "desc"
		}
	}
	if orderDirection == "asc" {
		operator = ">"
	}
	filter.OrderBy = orderBy
	filter.OrderDirection = orderDirection
	filter.Count = count
	filter.operator = operator
	filter.cursor = cursor

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

	nextToken, err := controller.EncodeToken(idioms, filter, false)
	previousToken, err := controller.EncodeToken(idioms, filter, true)
	body.Idioms = idioms
	body.Cursor.Previous = previousToken
	body.Cursor.Next = nextToken
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

	nextToken, err := controller.EncodeToken(idioms, filter, false)
	previousToken, err := controller.EncodeToken(idioms, filter, true)
	body.Idioms = idioms
	body.Cursor.Previous = previousToken
	body.Cursor.Next = nextToken
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

	nextToken, err := controller.EncodeToken(idioms, filter, false)
	previousToken, err := controller.EncodeToken(idioms, filter, true)
	body.Idioms = idioms
	body.Cursor.Previous = previousToken
	body.Cursor.Next = nextToken
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
		controller.logger.Println("Failed to parse form")
		controller.logger.PrintError("Error: %s", err)
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
		controller.logger.Println("Failed to get form file with id %s", idiomId)
		controller.logger.Println("Error: %s", err)
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
		controller.logger.Println("Failed to get body with id %s", id)
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
