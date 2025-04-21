package data

// Filters defines the parameters for paginating and sorting query results.
// It is used to control which page of results to return, how many results per page,
// and the field by which to sort the results.
type Filters struct {
	Page     int    // The current page number (1-based).
	PageSize int    // The number of items to return per page.
	Sort     string // The field to sort the results by (e.g., "id", "title").
}
