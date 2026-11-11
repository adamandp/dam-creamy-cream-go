package health

import (
	"database/sql"
	"net/http"
	"time"
)

type Service struct {
	db     *sql.DB
	client *http.Client
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, client: &http.Client{Timeout: 3 * time.Second}}
}
