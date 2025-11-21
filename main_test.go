package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestDB(t *testing.T) {
	// Setup
	dbName := "test_pantry.db"
	os.Remove(dbName) // Clean up previous runs
	InitDB(dbName)
	defer func() {
		DB.Close()
		os.Remove(dbName)
	}()

	// Test CreateItem
	item := Item{
		Name:           "Test Item",
		Description:    "A test item",
		ExpirationDate: time.Now().AddDate(0, 0, 2), // Expires in 2 days
		ImagePath:      "uploads/test.jpg",
	}

	err := CreateItem(item)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// Test GetItemsNearExpiration
	items, err := GetItemsNearExpiration(3) // Check for items expiring in 3 days
	if err != nil {
		t.Fatalf("Failed to get items: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	if items[0].Name != "Test Item" {
		t.Errorf("Expected item name 'Test Item', got '%s'", items[0].Name)
	}

	// Test GetItemByID
	fetchedItem, err := GetItemByID(items[0].ID)
	if err != nil {
		t.Fatalf("Failed to get item by ID: %v", err)
	}
	if fetchedItem.Name != "Test Item" {
		t.Errorf("Expected item name 'Test Item', got '%s'", fetchedItem.Name)
	}

	// Test GetAllItems
	allItems, err := GetAllItems()
	if err != nil {
		t.Fatalf("Failed to get all items: %v", err)
	}
	if len(allItems) != 1 {
		t.Errorf("Expected 1 item, got %d", len(allItems))
	}

	// Test List Handler
	req, err := http.NewRequest("GET", "/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleListItems)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Test MarkAsNotified
	err = MarkAsNotified(items[0].ID)
	if err != nil {
		t.Fatalf("Failed to mark as notified: %v", err)
	}

	// Verify it's not returned again
	items, err = GetItemsNearExpiration(3)
	if err != nil {
		t.Fatalf("Failed to get items: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items after notification, got %d", len(items))
	}
}
