package data

import "greenlight.tomcat.net/internal/validator"

// Filters defines the parameters for paginating and sorting query results.
// It is used to control which page of results to return, how many results per page,
// and the field by which to sort the results.
type Filters struct {
	Page         int      // The page number to retrieve (starts at 1).
	PageSize     int      // The maximum number of items to return per page.
	Sort         string   // The column or field to sort by (e.g., "id", "title", "-year").
	SortSafelist []string // List of permitted sort values to prevent unsafe input.
}

// ValidateFilters checks the Filters struct fields for valid values and records any validation errors.
func ValidateFilters(v *validator.Validator, f Filters) {
	// Check that the page number is greater than zero.
	v.Check(f.Page > 0, "page", "must be greater than zero")
	// Check that the page number does not exceed 10 million.
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	// Check that the page size is greater than zero.
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	// Check that the page size does not exceed 100.
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	// Check that the sort value is in the permitted safelist.
	v.Check(validator.PermittedValue(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}
