package main

import (
	"database/sql"
	"log"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

type Item struct {
	ID             int
	Name           string
	Description    string
	ExpirationDate time.Time
	ImagePath      string
	Notified       bool
	CreatedAt      time.Time
}

func (i Item) ThumbnailPath() string {
	if i.ImagePath == "" {
		return ""
	}
	dir, file := filepath.Split(i.ImagePath)
	return filepath.Join(dir, "thumb_"+file)
}

func InitDB(filepath string) {
	var err error
	DB, err = sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS items (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"name" TEXT,
		"description" TEXT,
		"expiration_date" DATETIME,
		"image_path" TEXT,
		"notified" BOOLEAN DEFAULT 0,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	createReceiversTableSQL := `CREATE TABLE IF NOT EXISTS receivers (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"email" TEXT NOT NULL UNIQUE
	);`

	_, err = DB.Exec(createReceiversTableSQL)
	if err != nil {
		log.Fatal(err)
	}
}

func CreateItem(item Item) error {
	insertSQL := `INSERT INTO items (name, description, expiration_date, image_path) VALUES (?, ?, ?, ?)`
	statement, err := DB.Prepare(insertSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec(item.Name, item.Description, item.ExpirationDate, item.ImagePath)
	return err
}

func GetItemsNearExpiration(days int) ([]Item, error) {
	query := `SELECT id, name, description, expiration_date, image_path, notified, created_at FROM items WHERE expiration_date <= ? AND notified = 0`

	targetDate := time.Now().AddDate(0, 0, days)
	rows, err := DB.Query(query, targetDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var i Item
		err = rows.Scan(&i.ID, &i.Name, &i.Description, &i.ExpirationDate, &i.ImagePath, &i.Notified, &i.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}

func MarkAsNotified(id int) error {
	updateSQL := `UPDATE items SET notified = 1 WHERE id = ?`
	statement, err := DB.Prepare(updateSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec(id)
	return err
}

func GetItemByID(id int) (*Item, error) {
	query := `SELECT id, name, description, expiration_date, image_path, notified, created_at FROM items WHERE id = ?`
	row := DB.QueryRow(query, id)

	var i Item
	err := row.Scan(&i.ID, &i.Name, &i.Description, &i.ExpirationDate, &i.ImagePath, &i.Notified, &i.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func GetAllItems() ([]Item, error) {
	query := `SELECT id, name, description, expiration_date, image_path, notified, created_at FROM items ORDER BY expiration_date ASC`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var i Item
		err = rows.Scan(&i.ID, &i.Name, &i.Description, &i.ExpirationDate, &i.ImagePath, &i.Notified, &i.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, nil
}

// Receiver struct
type Receiver struct {
	ID    int
	Email string
}

// AddReceiver adds a new email receiver
func AddReceiver(email string) error {
	insertSQL := `INSERT INTO receivers (email) VALUES (?)`
	statement, err := DB.Prepare(insertSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec(email)
	return err
}

// GetReceivers returns all email receivers
func GetReceivers() ([]Receiver, error) {
	query := `SELECT id, email FROM receivers`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receivers []Receiver
	for rows.Next() {
		var r Receiver
		err = rows.Scan(&r.ID, &r.Email)
		if err != nil {
			return nil, err
		}
		receivers = append(receivers, r)
	}
	return receivers, nil
}

// DeleteReceiver deletes a receiver by ID
func DeleteReceiver(id int) error {
	deleteSQL := `DELETE FROM receivers WHERE id = ?`
	statement, err := DB.Prepare(deleteSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec(id)
	return err
}
