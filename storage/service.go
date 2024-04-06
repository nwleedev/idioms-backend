package storage

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type StorageService interface {
	GetStorage() *s3.Client
}

type ServiceOption struct {
	roleArn         string
	roleSessionName string

	createdAt time.Time
	duration  time.Duration
}

type Service struct {
	config    *aws.Config
	s3Client  *s3.Client
	stsClient *sts.Client
	option    *ServiceOption
}

func NewService(config *aws.Config, awsId string, awsKey string, awsRoleArn string) *Service {
	config.Credentials = credentials.NewStaticCredentialsProvider(awsId, awsKey, "")
	stsClient := sts.NewFromConfig(*config)
	roleArn := awsRoleArn
	roleSessionName := "austin-idioms-sessions"

	service := new(Service)
	service.config = config
	service.stsClient = stsClient
	service.option = &ServiceOption{
		roleArn:         roleArn,
		roleSessionName: roleSessionName,
		createdAt:       time.Now(),
		duration:        time.Duration(time.Second * 900),
	}

	return service
}

func (service *Service) GetStorage() *s3.Client {
	if time.Since(service.option.createdAt) < time.Duration(time.Second*600) && service.s3Client != nil {
		return service.s3Client
	}
	credentials := stscreds.NewAssumeRoleProvider(service.stsClient, service.option.roleArn, func(roleOptions *stscreds.AssumeRoleOptions) {
		roleOptions.RoleSessionName = service.option.roleSessionName
		roleOptions.Duration = *aws.Duration((service.option.duration))
	})
	service.config.Credentials = aws.NewCredentialsCache(credentials)
	s3Client := s3.NewFromConfig(*service.config)

	service.s3Client = s3Client
	return service.s3Client
}
