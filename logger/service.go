package logger

import (
	"fmt"
	"log"
)

type LoggerService interface {
	Println(format string, v ...any)
	PrintError(format string, err error)
}

type Service struct {
	logger *log.Logger
}

func NewService(logger *log.Logger) LoggerService {
	service := new(Service)
	service.logger = logger
	return service
}

func (service *Service) Println(format string, v ...any) {
	if len(v) == 0 {
		service.logger.Println(format)
	}
	row := fmt.Sprintf(format, v...)
	service.logger.Printf("%s\n", row)
}

func (service *Service) PrintError(format string, err error) {
	prefix := format
	if format == "" {
		prefix = "Error: "
	}
	service.logger.Printf("%s %v\n", prefix, err)
}
