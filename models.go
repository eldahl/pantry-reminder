package main

import (
	"database/sql"
	"log"
	"path/filepath"
	"sort"
	"strings"
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
	ReminderDays   int
	Tags           []string
	CreatedAt      time.Time
}

func truncateToDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func (i Item) IsExpired() bool {
	return truncateToDate(i.ExpirationDate).Before(truncateToDate(time.Now()))
}

func (i Item) IsExpiringSoon() bool {
	if i.IsExpired() {
		return false
	}
	now := truncateToDate(time.Now())
	threshold := now.AddDate(0, 0, i.ReminderDays)
	return !truncateToDate(i.ExpirationDate).After(threshold)
}

func (i Item) ThumbnailPath() string {
	if i.ImagePath == "" {
		return ""
	}
	dir, file := filepath.Split(i.ImagePath)
	return filepath.Join(dir, "thumb_"+file)
}

func (i Item) SortTag() string {
	if len(i.Tags) == 0 {
		return ""
	}
	return strings.ToLower(i.Tags[0])
}

func NormalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, tag)
	}
	sort.Strings(normalized)
	return normalized
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

	if _, err = DB.Exec("PRAGMA foreign_keys = ON"); err != nil {
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

	// Migration: Add reminder_days column if it doesn't exist
	// We can just try to add it and ignore the error if it fails (simplest for SQLite)
	// Or check pragma. Let's just try to add it.
	alterTableSQL := `ALTER TABLE items ADD COLUMN reminder_days INTEGER DEFAULT 30;`
	_, err = DB.Exec(alterTableSQL)
	if err != nil {
		// Ignore error, likely column already exists
		log.Println("Migration warning (safe to ignore if column exists):", err)
	}

	alterTagSQL := `ALTER TABLE items ADD COLUMN tag TEXT DEFAULT '';`
	_, err = DB.Exec(alterTagSQL)
	if err != nil {
		log.Println("Migration warning (safe to ignore if column exists):", err)
	}

	createItemTagsSQL := `CREATE TABLE IF NOT EXISTS item_tags (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"item_id" integer NOT NULL,
		"tag" TEXT NOT NULL,
		FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
		UNIQUE(item_id, tag)
	);`
	_, err = DB.Exec(createItemTagsSQL)
	if err != nil {
		log.Fatal(err)
	}

	migrateLegacyTags()
}

func migrateLegacyTags() {
	rows, err := DB.Query(`SELECT id, tag FROM items WHERE tag IS NOT NULL AND tag != ''`)
	if err != nil {
		log.Println("Legacy tag migration warning:", err)
		return
	}
	defer rows.Close()

	insertSQL := `INSERT OR IGNORE INTO item_tags (item_id, tag) VALUES (?, ?)`
	statement, err := DB.Prepare(insertSQL)
	if err != nil {
		log.Println("Legacy tag migration warning:", err)
		return
	}
	defer statement.Close()

	for rows.Next() {
		var itemID int
		var legacyTag string
		if err := rows.Scan(&itemID, &legacyTag); err != nil {
			continue
		}
		for _, part := range strings.Split(legacyTag, ",") {
			tag := strings.TrimSpace(part)
			if tag == "" {
				continue
			}
			statement.Exec(itemID, tag)
		}
	}
}

func SetItemTags(itemID int, tags []string) error {
	tags = NormalizeTags(tags)

	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	if _, err = tx.Exec(`DELETE FROM item_tags WHERE item_id = ?`, itemID); err != nil {
		tx.Rollback()
		return err
	}

	insertSQL := `INSERT INTO item_tags (item_id, tag) VALUES (?, ?)`
	statement, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer statement.Close()

	for _, tag := range tags {
		if _, err = statement.Exec(itemID, tag); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func GetTagsForItem(itemID int) ([]string, error) {
	query := `SELECT tag FROM item_tags WHERE item_id = ? ORDER BY tag COLLATE NOCASE`
	rows, err := DB.Query(query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func attachTagsToItems(items []Item) error {
	if len(items) == 0 {
		return nil
	}

	tagMap := make(map[int][]string)
	rows, err := DB.Query(`SELECT item_id, tag FROM item_tags ORDER BY tag COLLATE NOCASE`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var itemID int
		var tag string
		if err := rows.Scan(&itemID, &tag); err != nil {
			return err
		}
		tagMap[itemID] = append(tagMap[itemID], tag)
	}

	for i := range items {
		items[i].Tags = tagMap[items[i].ID]
		if items[i].Tags == nil {
			items[i].Tags = []string{}
		}
	}
	return nil
}

func CreateItem(item Item) (int, error) {
	insertSQL := `INSERT INTO items (name, description, expiration_date, image_path, reminder_days) VALUES (?, ?, ?, ?, ?)`
	statement, err := DB.Prepare(insertSQL)
	if err != nil {
		return 0, err
	}
	result, err := statement.Exec(item.Name, item.Description, item.ExpirationDate, item.ImagePath, item.ReminderDays)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := SetItemTags(int(id), item.Tags); err != nil {
		return 0, err
	}

	return int(id), nil
}

func UpdateItem(item Item) error {
	updateSQL := `UPDATE items SET name = ?, description = ?, expiration_date = ?, image_path = ?, reminder_days = ?, notified = ? WHERE id = ?`
	statement, err := DB.Prepare(updateSQL)
	if err != nil {
		return err
	}
	if _, err = statement.Exec(item.Name, item.Description, item.ExpirationDate, item.ImagePath, item.ReminderDays, item.Notified, item.ID); err != nil {
		return err
	}
	return SetItemTags(item.ID, item.Tags)
}

func GetItemsNearExpiration() ([]Item, error) {
	// SQLite specific date math
	query := `SELECT id, name, description, expiration_date, image_path, notified, reminder_days, created_at FROM items WHERE expiration_date <= DATE('now', '+' || reminder_days || ' days') AND notified = 0`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var i Item
		err = rows.Scan(&i.ID, &i.Name, &i.Description, &i.ExpirationDate, &i.ImagePath, &i.Notified, &i.ReminderDays, &i.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := attachTagsToItems(items); err != nil {
		return nil, err
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
	query := `SELECT id, name, description, expiration_date, image_path, notified, reminder_days, created_at FROM items WHERE id = ?`
	row := DB.QueryRow(query, id)

	var i Item
	err := row.Scan(&i.ID, &i.Name, &i.Description, &i.ExpirationDate, &i.ImagePath, &i.Notified, &i.ReminderDays, &i.CreatedAt)
	if err != nil {
		return nil, err
	}

	i.Tags, err = GetTagsForItem(id)
	if err != nil {
		return nil, err
	}
	if i.Tags == nil {
		i.Tags = []string{}
	}
	return &i, nil
}

func GetAllItems() ([]Item, error) {
	query := `SELECT id, name, description, expiration_date, image_path, notified, reminder_days, created_at FROM items`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var i Item
		err = rows.Scan(&i.ID, &i.Name, &i.Description, &i.ExpirationDate, &i.ImagePath, &i.Notified, &i.ReminderDays, &i.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := attachTagsToItems(items); err != nil {
		return nil, err
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

// DeleteItem deletes an item by ID
func DeleteItem(id int) error {
	deleteSQL := `DELETE FROM items WHERE id = ?`
	statement, err := DB.Prepare(deleteSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec(id)
	return err
}
