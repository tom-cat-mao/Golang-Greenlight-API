package data

import (
	"context"
	"database/sql"
	"slices"
	"time"
)

// Define a Permissions slice, which we will use to hold the permission codes
type Permissions []string

// Include checks if the Permissions slice contains a specific permission code.
// It returns true if the code is found, false otherwise.
func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

// Define the PermissionModel type
type PermissionModel struct {
	DB *sql.DB
}

// GetAllForUser retrieves all permission codes associated with a specific user ID.
// It performs a join across the `permissions`, `users_permissions`, and `users` tables
// to find the permission codes linked to the given user.
// Returns:
// - Permissions: A slice of strings containing the permission codes.
// - error: Any database error encountered during the operation.
func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = $1
		`

	// Create a context with a 3-second timeout to prevent long-running database operations.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// Ensure the context is cancelled to free up resources once the operation completes.
	defer cancel()

	// Execute the query with the user ID as a parameter.
	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		// Return any error encountered during query execution.
		return nil, err
	}
	// Ensure the result set is closed after processing to free up database resources.
	defer rows.Close()

	// Initialize an empty slice to hold the permission codes.
	var permissions Permissions

	// Iterate over the rows in the result set.
	for rows.Next() {
		var permission string

		// Scan the permission code from the current row into the 'permission' variable.
		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		// Append the scanned permission code to the 'permissions' slice.
		permissions = append(permissions, permission)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}
