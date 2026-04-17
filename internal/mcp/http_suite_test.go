package mcp

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// HTTPTestSuite is a test suite for HTTP transport tests
type HTTPTestSuite struct {
	suite.Suite
}

func TestHTTPSuite(t *testing.T) {
	suite.Run(t, new(HTTPTestSuite))
}
