package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/nw.lee/idioms-backend/idioms"
	"github.com/nw.lee/idioms-backend/logger"
)

type Handler struct {
	idiomController idioms.IdiomController
	router          *chi.Mux
	logger          logger.LoggerService

	isAdmin bool
}

func NewHandler(isAdmin bool) *Handler {
	handler := new(Handler)
	handler.router = chi.NewRouter()
	handler.logger = logger.NewService(log.Default())
	handler.isAdmin = isAdmin

	return handler
}

func (handler *Handler) AddIdiomController(controller idioms.IdiomController) *Handler {
	handler.idiomController = controller
	return handler
}

func (handler *Handler) Run() {

	handler.router.Use(middleware.Logger)
	handler.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://useidioms.com", "https://api.useidioms.com", "http://useidioms.com", "http://api.useidioms.com", "http://localhost:8082"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           600,
	}))
	handler.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			res.Header().Add("content-type", "application/json")
			next.ServeHTTP(res, req)
		})
	})
	handler.router.Get("/idioms/admin", handler.idiomController.GetIdioms)
	handler.router.Get("/idioms", handler.idiomController.GetIdiomsWithThumbnail)
	handler.router.Get("/idioms/{id}", handler.idiomController.GetIdiomById)
	handler.router.Get("/idioms/{id}/related", handler.idiomController.GetRelatedIdioms)
	handler.router.Get("/idioms/search", handler.idiomController.SearchIdioms)

	if handler.isAdmin {
		handler.router.Post("/idioms/inputs", handler.idiomController.CreateIdiomInputs)
		handler.router.Post("/idioms/thumbnail/draft", handler.idiomController.CreateThumbnail)
		handler.router.Post("/idioms/thumbnail/file", handler.idiomController.UploadThumbnail)
		handler.router.Post("/idioms/thumbnail/url", handler.idiomController.CreateThumbnailByURL)
		handler.router.Post("/idioms/{id}/thumbnail", handler.idiomController.UpdateThumbnailPrompt)
	}
}

func (handler *Handler) ListenAndServe(address string) {
	http.ListenAndServe(address, handler.router)
}
