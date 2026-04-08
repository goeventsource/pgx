// Package pgxtest provides utility functions for easier testing of PostgreSQL-related functionality,
// aiming to improve the developer experience when working with PostgreSQL databases.
//
// The primary goal of the package is to simplify the process of creating seeded database connections for testing purposes.
// It provides functions that start a PostgreSQL container, perform necessary setup,
// and return a *pgxpool.Pool ready to be used in tests or demos.
// Build on top of that, there are other helper functions such as NewStore and NewRepository.
// This approach ensures a consistent and controlled environment for testing PostgreSQL-related functionality.
package pgxtest
