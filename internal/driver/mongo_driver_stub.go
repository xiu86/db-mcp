//go:build nomongo
// +build nomongo

package driver

import (
	"db-mcp/internal/config"
	"db-mcp/pkg/logger"
)

// NewMongoDriver creates a MongoDB driver stub when nomongo build tag is set
func NewMongoDriver(cfg *config.MongoConfig, log *logger.Logger) (DatabaseDriver, error) {
	return nil, &MongoNotAvailableError{}
}

type MongoNotAvailableError struct{}

func (e *MongoNotAvailableError) Error() string {
	return "MongoDB driver not available - build without 'nomongo' tag to enable"
}
