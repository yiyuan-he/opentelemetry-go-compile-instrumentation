package main

import (
	"database/sql"
	"fmt"
	"net/http"
	_ "github.com/mattn/go-sqlite3"
)

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./demo.db")
	if err != nil {
			return nil, err
	}

	// Create table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT
	)`)
	if err != nil {
			return nil, err
	}

	// Add sample data if table is empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
			return nil, err
	}

	if count == 0 {
			_, err = db.Exec(`INSERT INTO users (name) VALUES
					('Alice'),
					('Bob'),
					('Charlie')`)
			if err != nil {
					return nil, err
			}
	}

	return db, nil
}

func queryUsers(w http.ResponseWriter, r *http.Request) {
	db, err := initDB()
	if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
	}
	defer db.Close()

	rows, err := db.QueryContext(r.Context(), "SELECT id, name FROM users")
	if err != nil {
			http.Error(w, "Query error: "+err.Error(), http.StatusInternalServerError)
			return
	}
	defer rows.Close()

	fmt.Fprintf(w, "Users:\n")
	for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
					http.Error(w, "Scan error", http.StatusInternalServerError)
					return
			}
			fmt.Fprintf(w, "ID: %d, Name: %s\n", id, name)
	}
}
