package data

import (
	"strings"

	"greenlight.tomcat.net/internal/validator"
)

// Filters defines the parameters for paginating and sorting query results.
// It is used to control which page of results to return, how many results per page,
// and the field by which to sort the results.
type Filters struct {
	Page         int      // The page number to retrieve (starts at 1).
	PageSize     int      // The maximum number of items to return per page.
	Sort         string   // The column or field to sort by (e.g., "id", "title", "-year").
	SortSafelist []string // List of permitted sort values to prevent unsafe input.
}

// sortColumn returns the column name to use for sorting, after validating that the requested sort value
// is present in the SortSafelist. If the sort value is not permitted, it panics to prevent unsafe SQL injection.
func (f Filters) sortColumn() string {
	// Iterate through the safelist of permitted sort values.
	for _, safeValue := range f.SortSafelist {
		// If the requested sort value matches a permitted value,
		// return the column name with any leading '-' (for descending order) removed.
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}
	// Panic if the sort value is not in the safelist, indicating an unsafe or invalid sort parameter.
	panic("unsafe sort parameter: " + f.Sort)
}

// sortDirection returns the SQL sort direction ("ASC" or "DESC") based on the Filters.Sort value.
// If the sort value starts with a '-', it indicates descending order ("DESC").
// Otherwise, it defaults to ascending order ("ASC").
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
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

// limit returns the maximum number of items to retrieve per page for pagination.
// It simply returns the PageSize field from the Filters struct.
func (f Filters) limit() int {
	return f.PageSize
}

// offset returns the number of items to skip for pagination based on the current page and page size.
// It calculates the offset as (Page - 1) * PageSize, which is used in SQL queries to determine
// where to start returning results for the requested page.
func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}
