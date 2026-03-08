package config

import "errors"

var (
	// ErrRepositoryExists is returned when a repository already exists
	ErrRepositoryExists = errors.New("repository already exists")

	// ErrRepositoryNotFound is returned when a repository is not found
	ErrRepositoryNotFound = errors.New("repository not found")
)
