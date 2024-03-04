package testutils

import (
	"embed"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

const (
	Uuidv4Regex  = "[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89aAbB][a-f0-9]{3}-[a-f0-9]{12}"
	RootUrl      = "http://fakeurl:3001/api/v1/"
	Neighborhood = "1"
)

var (
	DefaultMigrationsUrl = RootUrl + fmt.Sprintf("neighborhoods/%s/migrations/", "0")
	DefaultMigrationUrl  = DefaultMigrationsUrl + Uuidv4Regex

	DefaultTransactionUrl = RootUrl + fmt.Sprintf("neighborhoods/%s/transactions/", "0")

	MigrationsUrl = RootUrl + fmt.Sprintf("neighborhoods/%s/migrations/", Neighborhood)
	MigrationUrl  = MigrationsUrl + Uuidv4Regex
	LoginUrl      = RootUrl + "auth/login"
)

//go:embed testdata/*
var mockData embed.FS

// CreateJsonResponderFromFile creates a new JSON responder from a file.
func CreateJsonResponderFromFile(filePath string, code int) (httpmock.Responder, error) {
	inputData, err := mockData.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var jsonData interface{}
	if err = json.Unmarshal(inputData, &jsonData); err != nil {
		return nil, err
	}

	return httpmock.NewJsonResponder(code, jsonData)
}

// SetupMockResponder sets up a mock responder for a given method and URL.
func SetupMockResponder(t *testing.T, method, url, filePath string, code int) {
	var responder httpmock.Responder
	var err error
	if filePath != "" {
		responder, err = CreateJsonResponderFromFile(filePath, code)
		require.NoError(t, err)
	} else {
		responder = httpmock.NewErrorResponder(fmt.Errorf("not found"))
	}

	httpmock.RegisterResponder(method, url, responder)
}

type Endpoint struct {
	Method string
	Url    string
	Data   string
	Code   int
}
