package search

import "github.com/hyperjump/sagasu/internal/models"

// ProcessQuery validates and applies defaults to the search query.
func ProcessQuery(query *models.SearchQuery) error {
	return query.Validate()
}
