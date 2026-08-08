package server

import (
	"encoding/json"
	"os"

	"github.com/amavis442/mockserver/internal/domain"
	"github.com/amavis442/mockserver/internal/engine"
)

// LoadConfig reads a JSON file containing an array of expectations and adds
// them to the store via Upsert. Any existing expectations in the store are
// preserved. The file format is the same as the POST /__admin/expectations
// request body — an array of Expectation objects.
func LoadConfig(store *engine.Store, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var expectations []domain.Expectation
	if err := json.Unmarshal(data, &expectations); err != nil {
		return err
	}

	for _, exp := range expectations {
		store.Upsert(exp)
	}
	return nil
}
