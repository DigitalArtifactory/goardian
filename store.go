package main

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"
)

type Service struct {
	ID                 string
	Name               string
	Method             string
	Endpoint           string
	Payload            string
	RequestDelay       string // Milliseconds
	JSONProperty       string
	PreferredStatus    string
	InsecureSkipVerify string // Boolean (Y, N)
	// Non column values
	LastStatusInfo string
	StatusHistory  []bool
}

type Store struct {
	conn *sql.DB
}

func (s *Store) Init() error {
	var err error
	s.conn, err = sql.Open("sqlite3", "./goardian.db")
	if err != nil {
		return err
	}

	createTableStmt := `CREATE TABLE IF NOT EXISTS services (
		id text not null primary key,
		name text not null,
		method text not null,
		endpoint text not null,
		payload text not null,
		request_delay text not null,
		json_property text null,
		preferred_status text null,
		insecure_skip_verify text null
	);`

	if _, err := s.conn.Exec(createTableStmt); err != nil {
		return err
	}

	return nil
}

func (s *Store) GetServices() ([]Service, error) {
	rows, err := s.conn.Query("SELECT * FROM services")
	if err != nil {
		return nil, err
	}

	services := []Service{}
	defer rows.Close()
	for rows.Next() {
		service := Service{}
		rows.Scan(&service.ID, &service.Name, &service.Method, &service.Endpoint, &service.Payload, &service.RequestDelay, &service.JSONProperty, &service.PreferredStatus, &service.InsecureSkipVerify)
		services = append(services, service)
	}

	return services, nil
}

func (s *Store) SaveService(service Service) error {
	if service.ID == "" {
		id := uuid.New()
		service.ID = id.String()
	}

	upsertQuery := `INSERT INTO services (id, name, method, endpoint, payload, request_delay, json_property, preferred_status, insecure_skip_verify)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE
	SET name=excluded.name, method=excluded.method, endpoint=excluded.endpoint, payload=excluded.payload, request_delay=excluded.request_delay, json_property=excluded.json_property, preferred_status=excluded.preferred_status, insecure_skip_verify=excluded.insecure_skip_verify;`

	if _, err := s.conn.Exec(upsertQuery, service.ID, service.Name, service.Method, service.Endpoint, service.Payload, service.RequestDelay, service.JSONProperty, service.PreferredStatus, service.InsecureSkipVerify); err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteService(service Service) error {
	if service.ID == "" {
		// if not exists throw an error
		err := fmt.Errorf("could not find service %v with ID: %v", service.Name, service.ID)
		return err
	}

	deleteQuery := `DELETE FROM services WHERE id = ?;`

	if _, err := s.conn.Exec(deleteQuery, service.ID); err != nil {
		return err
	}

	return nil
}
