package main

import (
	"database/sql"
	"fmt"
	"os"

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
	ExpectedValue      string
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

	// Check if goardian.db exists and backup if needed
	if err := s.backupExistingDatabase(); err != nil {
		return fmt.Errorf("failed to backup existing database: %w", err)
	}

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
		expected_value text null,
		preferred_status text null,
		insecure_skip_verify text null
	);`

	if _, err := s.conn.Exec(createTableStmt); err != nil {
		return err
	}
	createHistoryTableStmt := `CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id TEXT NOT NULL,
		status BOOLEAN NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(service_id) REFERENCES services(id)
	);`

	if _, err := s.conn.Exec(createHistoryTableStmt); err != nil {
		return err
	}

	// Restore data from backup if it exists
	if err := s.restoreFromBackup(); err != nil {
		return fmt.Errorf("failed to restore data from backup: %w", err)
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
		rows.Scan(&service.ID, &service.Name, &service.Method, &service.Endpoint, &service.Payload, &service.RequestDelay, &service.JSONProperty, &service.ExpectedValue, &service.PreferredStatus, &service.InsecureSkipVerify)
		services = append(services, service)
		for i := range services {
			historyRows, err := s.conn.Query(
				"SELECT status FROM history WHERE service_id = ? ORDER BY timestamp DESC LIMIT 20",
				services[i].ID,
			)
			if err == nil {
				defer historyRows.Close()
				services[i].StatusHistory = []bool{}
				for historyRows.Next() {
					var status bool
					if err := historyRows.Scan(&status); err == nil {
						services[i].StatusHistory = append(services[i].StatusHistory, status)
					}
				}
			}
		}

	}

	return services, nil
}

func (s *Store) SaveService(service Service) error {
	if service.ID == "" {
		id := uuid.New()
		service.ID = id.String()
	}

	upsertQuery := `INSERT INTO services (id, name, method, endpoint, payload, request_delay, json_property, expected_value, preferred_status, insecure_skip_verify)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE
	SET name=excluded.name, method=excluded.method, endpoint=excluded.endpoint, payload=excluded.payload, request_delay=excluded.request_delay, json_property=excluded.json_property, expected_value=excluded.expected_value, preferred_status=excluded.preferred_status, insecure_skip_verify=excluded.insecure_skip_verify;`

	if _, err := s.conn.Exec(upsertQuery, service.ID, service.Name, service.Method, service.Endpoint, service.Payload, service.RequestDelay, service.JSONProperty, service.ExpectedValue, service.PreferredStatus, service.InsecureSkipVerify); err != nil {
		return err
	}

	return nil
}

func (s *Store) SaveHistory(service Service, status bool) error {
	insertQuery := `INSERT INTO history (service_id, status) VALUES (?, ?);`
	if _, err := s.conn.Exec(insertQuery, service.ID, status); err != nil {
		return err
	}
	return nil
}

func (s *Store) DeleteAllHistory(service Service) error {
	deleteQuery := `DELETE FROM history WHERE service_id = ?;`
	if _, err := s.conn.Exec(deleteQuery, service.ID); err != nil {
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

	// Delete service
	deleteQuery := `DELETE FROM services WHERE id = ?;`
	if _, err := s.conn.Exec(deleteQuery, service.ID); err != nil {
		return err
	}

	// Delete history
	if err := s.DeleteAllHistory(service); err != nil {
		return err
	}

	return nil
}

// backupExistingDatabase checks if goardian.db exists and renames it to goardian.bak.db
func (s *Store) backupExistingDatabase() error {
	dbPath := "./goardian.db"
	backupPath := "./goardian.bak.db"

	// Check if the database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Database doesn't exist, nothing to backup
		return nil
	}

	// Remove existing backup if it exists
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("failed to remove existing backup: %w", err)
		}
	}

	// Rename current database to backup
	if err := os.Rename(dbPath, backupPath); err != nil {
		return fmt.Errorf("failed to rename database to backup: %w", err)
	}

	return nil
}

// restoreFromBackup restores data from the backup database if it exists
func (s *Store) restoreFromBackup() error {
	backupPath := "./goardian.bak.db"

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		// No backup to restore from
		return nil
	}

	// Open backup database
	backupConn, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}
	defer backupConn.Close()

	// Get all services from backup
	rows, err := backupConn.Query("SELECT * FROM services")
	if err != nil {
		// If the table doesn't exist in backup, that's okay
		return nil
	}
	defer rows.Close()

	// Restore each service to the new database
	for rows.Next() {
		var service Service
		err := rows.Scan(&service.ID, &service.Name, &service.Method, &service.Endpoint,
			&service.Payload, &service.RequestDelay, &service.JSONProperty,
			&service.ExpectedValue, &service.PreferredStatus, &service.InsecureSkipVerify)
		if err != nil {
			return fmt.Errorf("failed to scan service from backup: %w", err)
		}

		// Insert service into new database
		if err := s.SaveService(service); err != nil {
			return fmt.Errorf("failed to restore service %s: %w", service.Name, err)
		}
	}

	// Restore history from backup
	historyRows, err := backupConn.Query("SELECT service_id, status, timestamp FROM history")
	if err == nil {
		defer historyRows.Close()
		for historyRows.Next() {
			var serviceID string
			var status bool
			var timestamp string
			if err := historyRows.Scan(&serviceID, &status, &timestamp); err != nil {
				return fmt.Errorf("failed to scan history from backup: %w", err)
			}
			_, err := s.conn.Exec(
				`INSERT INTO history (service_id, status, timestamp) VALUES (?, ?, ?)`,
				serviceID, status, timestamp,
			)
			if err != nil {
				return fmt.Errorf("failed to restore history for service %s: %w", serviceID, err)
			}
		}
	}

	return nil
}
