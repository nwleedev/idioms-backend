package logger

import (
	"fmt"
	"log"
	"runtime"
	"time"
)

type LoggerService interface {
	Info(values ...any)
	Warn(message string, values ...any)
	Error(err error, message string, values ...any)
}

type Service struct {
	logger *log.Logger
}

func NewService(logger *log.Logger) LoggerService {
	service := new(Service)
	service.logger = logger
	return service
}

func (service *Service) Info(values ...any) {
	runtimeCaller, _, _, _ := runtime.Caller(1)

	callerName := runtime.FuncForPC(runtimeCaller).Name()
	now := time.Now()
	createdAt := now.Format(time.DateTime)
	title := "INFO"
	infoBody := fmt.Sprintln(values...)

	log.Printf("%s %s %s %s\n", title, createdAt, callerName, infoBody)
}

func (service *Service) Warn(message string, values ...any) {
	runtimeCaller, _, _, _ := runtime.Caller(1)

	callerName := runtime.FuncForPC(runtimeCaller).Name()
	now := time.Now()
	createdAt := now.Format(time.DateTime)
	title := "WARN"
	warnMessage := fmt.Sprintln(values...)
	warnBody := fmt.Sprintf("---\nMessage: %s %s---", message, warnMessage)

	log.Printf("%s %s %s %s\n", title, createdAt, callerName, warnBody)
}

func (service *Service) Error(err error, message string, values ...any) {
	runtimeCaller, _, _, _ := runtime.Caller(1)

	callerName := runtime.FuncForPC(runtimeCaller).Name()
	now := time.Now()
	createdAt := now.Format(time.DateTime)
	title := "ERROR"
	errorMessage := fmt.Sprintln(values...)
	errorBody := fmt.Sprintf("\n---\nMessage: %s %s%s\n---", message, errorMessage, err)

	log.Printf("%s %s %s\n%s\n", title, createdAt, callerName, errorBody)
}
