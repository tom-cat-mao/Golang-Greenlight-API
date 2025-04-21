package data

import (
	"strings"

	"greenlight.tomcat.net/internal/validator"
)

// Metadata holds pagination information for API responses.
// It is typically included in responses that return a paginated list of resources
// to help clients understand the current page, page size, and total number of records.
type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`  // The current page number being returned.
	PageSize     int `json:"page_size,omitempty"`     // The number of records per page.
	FirstPage    int `json:"first_page,omitempty"`    // The first page number (usually 1).
	LastPage     int `json:"last_page,omitempty"`     // The last available page number.
	TotalRecords int `json:"total_records,omitempty"` // The total number of records matching the query.
}

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

// calculateMetadata computes pagination metadata for a paginated API response.
// It takes the total number of records, the current page, and the page size as input,
// and returns a Metadata struct containing information about the current page, page size,
// first page, last page, and total number of records.
func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	// If there are no records, return an empty Metadata struct.
	if totalRecords == 0 {
		return Metadata{}
	}

	// Calculate the last page number using integer division.
	// (totalRecords + pageSize - 1) / pageSize ensures that any partial page is counted as a full page.
	lastPage := (totalRecords + pageSize - 1) / pageSize

	// Return the populated Metadata struct.
	return Metadata{
		CurrentPage:  page,         // The current page number.
		PageSize:     pageSize,     // The number of records per page.
		FirstPage:    1,            // The first page is always 1.
		LastPage:     lastPage,     // The calculated last page number.
		TotalRecords: totalRecords, // The total number of records.
	}
}
