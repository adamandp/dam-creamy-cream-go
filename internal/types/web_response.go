// Package types defines shared application types.
package types

// Paging describes a page of results.
type Paging struct {
	CurrentPage int `json:"currentPage"`
	PageSize    int `json:"pageSize"`
	TotalItems  int `json:"totalItems"`
	TotalPages  int `json:"totalPages"`
}

// WebResponse wraps application data and errors in a consistent JSON envelope.
type WebResponse[T any] struct {
	Message string  `json:"message,omitempty"`
	Errors  string  `json:"errors,omitempty"`
	Data    T       `json:"data,omitempty"`
	Paging  *Paging `json:"paging,omitempty"`
}

type MessageResponse = WebResponse[*struct{}]
