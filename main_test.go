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
		ReminderDays:   30,
		Tags:           []string{"pantry", "test"},
	}

	_, err := CreateItem(item)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// Test GetItemsNearExpiration
	items, err := GetItemsNearExpiration() // Check for items expiring in 3 days
	if err != nil {
		t.Fatalf("Failed to get items: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	if items[0].Name != "Test Item" {
		t.Errorf("Expected item name 'Test Item', got '%s'", items[0].Name)
	}
	if len(items[0].Tags) != 2 || items[0].Tags[0] != "pantry" {
		t.Errorf("Expected item tags [pantry test], got %v", items[0].Tags)
	}

	// Test GetItemByID
	fetchedItem, err := GetItemByID(items[0].ID)
	if err != nil {
		t.Fatalf("Failed to get item by ID: %v", err)
	}
	if fetchedItem.Name != "Test Item" {
		t.Errorf("Expected item name 'Test Item', got '%s'", fetchedItem.Name)
	}

	updatedItem := *fetchedItem
	updatedItem.Name = "Updated Item"
	updatedItem.Description = "Updated description"
	updatedItem.Tags = []string{"snacks", "updated"}
	updatedItem.ReminderDays = 14
	err = UpdateItem(updatedItem)
	if err != nil {
		t.Fatalf("Failed to update item: %v", err)
	}

	updatedFetched, err := GetItemByID(fetchedItem.ID)
	if err != nil {
		t.Fatalf("Failed to get updated item: %v", err)
	}
	if updatedFetched.Name != "Updated Item" {
		t.Errorf("Expected updated name 'Updated Item', got '%s'", updatedFetched.Name)
	}
	if len(updatedFetched.Tags) != 2 || updatedFetched.Tags[0] != "snacks" {
		t.Errorf("Expected updated tags [snacks updated], got %v", updatedFetched.Tags)
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
	items, err = GetItemsNearExpiration()
	if err != nil {
		t.Fatalf("Failed to get items: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items after notification, got %d", len(items))
	}

	// Test Receivers
	err = AddReceiver("test@example.com")
	if err != nil {
		t.Fatalf("Failed to add receiver: %v", err)
	}

	receivers, err := GetReceivers()
	if err != nil {
		t.Fatalf("Failed to get receivers: %v", err)
	}
	if len(receivers) != 1 {
		t.Errorf("Expected 1 receiver, got %d", len(receivers))
	}
	if receivers[0].Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", receivers[0].Email)
	}

	err = DeleteReceiver(receivers[0].ID)
	if err != nil {
		t.Fatalf("Failed to delete receiver: %v", err)
	}

	receivers, err = GetReceivers()
	if err != nil {
		t.Fatalf("Failed to get receivers: %v", err)
	}
	if len(receivers) != 0 {
		t.Errorf("Expected 0 receivers, got %d", len(receivers))
	}
}

func TestItemExpirationStatus(t *testing.T) {
	expiredItem := Item{
		ExpirationDate: time.Now().AddDate(0, 0, -1),
		ReminderDays:   7,
	}
	if !expiredItem.IsExpired() {
		t.Error("Expected item to be expired")
	}
	if expiredItem.IsExpiringSoon() {
		t.Error("Expired item should not be expiring soon")
	}

	expiringItem := Item{
		ExpirationDate: time.Now().AddDate(0, 0, 3),
		ReminderDays:   7,
	}
	if expiringItem.IsExpired() {
		t.Error("Expected item not to be expired")
	}
	if !expiringItem.IsExpiringSoon() {
		t.Error("Expected item to be expiring soon")
	}

	freshItem := Item{
		ExpirationDate: time.Now().AddDate(0, 0, 30),
		ReminderDays:   7,
	}
	if freshItem.IsExpiringSoon() {
		t.Error("Fresh item should not be expiring soon")
	}
}

func TestSortItems(t *testing.T) {
	items := []Item{
		{Name: "Banana", Tags: []string{"fruit"}, ExpirationDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "Apple", Tags: []string{"dairy"}, ExpirationDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "Carrot", Tags: []string{"vegetable"}, ExpirationDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}

	sortItems(items, "name", "asc")
	if items[0].Name != "Apple" || items[2].Name != "Carrot" {
		t.Errorf("Unexpected name sort order: %s, %s, %s", items[0].Name, items[1].Name, items[2].Name)
	}

	sortItems(items, "tag", "desc")
	if items[0].SortTag() != "vegetable" || items[2].SortTag() != "dairy" {
		t.Errorf("Unexpected tag sort order: %s, %s, %s", items[0].SortTag(), items[1].SortTag(), items[2].SortTag())
	}

	sortItems(items, "date", "asc")
	if items[0].Name != "Apple" || items[2].Name != "Carrot" {
		t.Errorf("Unexpected date sort order: %s, %s, %s", items[0].Name, items[1].Name, items[2].Name)
	}
}

func TestFilterItems(t *testing.T) {
	items := []Item{
		{Name: "Expired", ExpirationDate: time.Now().AddDate(0, 0, -1)},
		{Name: "Fresh", ExpirationDate: time.Now().AddDate(0, 0, 10)},
	}

	expiredOnly := filterItems(items, true, false)
	if len(expiredOnly) != 1 || expiredOnly[0].Name != "Expired" {
		t.Errorf("Expected only expired item, got %v", expiredOnly)
	}

	nonExpiredOnly := filterItems(items, false, true)
	if len(nonExpiredOnly) != 1 || nonExpiredOnly[0].Name != "Fresh" {
		t.Errorf("Expected only non-expired item, got %v", nonExpiredOnly)
	}

	none := filterItems(items, false, false)
	if len(none) != 0 {
		t.Errorf("Expected no items when both filters disabled, got %d", len(none))
	}
}

func TestSearchItems(t *testing.T) {
	items := []Item{
		{Name: "Milk", Description: "Whole milk", Tags: []string{"dairy"}, ExpirationDate: time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)},
		{Name: "Bread", Description: "Sourdough loaf", Tags: []string{"bakery"}, ExpirationDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	byName := searchItems(items, "milk")
	if len(byName) != 1 || byName[0].Name != "Milk" {
		t.Errorf("Expected to find milk by name, got %v", byName)
	}

	byDescription := searchItems(items, "sourdough")
	if len(byDescription) != 1 || byDescription[0].Name != "Bread" {
		t.Errorf("Expected to find bread by description, got %v", byDescription)
	}

	byTag := searchItems(items, "dairy")
	if len(byTag) != 1 || byTag[0].Name != "Milk" {
		t.Errorf("Expected to find milk by tag, got %v", byTag)
	}

	byDate := searchItems(items, "2026-06-01")
	if len(byDate) != 1 || byDate[0].Name != "Bread" {
		t.Errorf("Expected to find bread by date, got %v", byDate)
	}

	noMatch := searchItems(items, "xyz")
	if len(noMatch) != 0 {
		t.Errorf("Expected no matches, got %d", len(noMatch))
	}
}

func TestNormalizeTags(t *testing.T) {
	tags := NormalizeTags([]string{" Dairy ", "frozen", "dairy", "", "Frozen"})
	if len(tags) != 2 || tags[0] != "Dairy" || tags[1] != "frozen" {
		t.Errorf("Unexpected normalized tags: %v", tags)
	}
}
