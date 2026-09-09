package gcp_services

import (
	"boilerplate-api/internal/config"
	"fmt"
	"path/filepath"

	"google.golang.org/api/option"
)

func NewGCPClientOption(logger config.Logger, env config.Env) option.ClientOption {
	serviceAccountKeyFilePath, err := filepath.Abs(fmt.Sprintf("./%v", env.ServiceAccountKey))
	if err != nil {
		logger.Panic("Unable to load serviceAccountKey.json file")
	}

	return option.WithCredentialsFile(serviceAccountKeyFilePath)
}
