package main

import (
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Initialize Database
	InitDB("pantry.db")
	defer DB.Close()

	// Start Background Job
	go startExpirationChecker()

	// HTTP Handlers
	http.HandleFunc("/", handleAddForm)
	http.HandleFunc("/add", handleAddItem)
	http.HandleFunc("/item", handleViewItem)
	http.HandleFunc("/edit", handleEditForm)
	http.HandleFunc("/edit-item", handleEditItem)
	http.HandleFunc("/list", handleListItems)
	http.HandleFunc("/settings", handleSettings)
	http.HandleFunc("/settings/add-receiver", handleAddReceiver)
	http.HandleFunc("/settings/delete-receiver", handleDeleteReceiver)
	http.HandleFunc("/delete-item", handleDeleteItem)
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Server started on :80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

func parseItemID(r *http.Request) (int, error) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		idStr = r.FormValue("id")
	}
	return strconv.Atoi(idStr)
}

type itemFormData struct {
	Name           string
	Description    string
	ExpirationDate time.Time
	ReminderDays   int
	Tags           []string
}

func parseItemForm(r *http.Request) (itemFormData, error) {
	expirationStr := r.FormValue("expiration_date")
	expirationDate, err := time.Parse("2006-01-02", expirationStr)
	if err != nil {
		return itemFormData{}, err
	}

	reminderDays := 30
	if reminderDaysStr := r.FormValue("reminder_days"); reminderDaysStr != "" {
		if rd, err := strconv.Atoi(reminderDaysStr); err == nil {
			reminderDays = rd
		}
	}

	return itemFormData{
		Name:           r.FormValue("name"),
		Description:    r.FormValue("description"),
		ExpirationDate: expirationDate,
		ReminderDays:   reminderDays,
		Tags:           NormalizeTags(r.Form["tags"]),
	}, nil
}

func saveUploadedImage(r *http.Request) (string, error) {
	file, handler, err := r.FormFile("image")
	if err != nil {
		return "", err
	}
	defer file.Close()

	os.MkdirAll("uploads", os.ModePerm)

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
	imagePath := filepath.Join("uploads", filename)
	dst, err := os.Create(imagePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	thumbPath := filepath.Join("uploads", "thumb_"+filename)
	if err := createThumbnail(imagePath, thumbPath); err != nil {
		log.Println("Error creating thumbnail:", err)
	}

	return imagePath, nil
}

func removeItemImages(item Item) {
	if item.ImagePath == "" {
		return
	}
	os.Remove(item.ImagePath)
	if thumbPath := item.ThumbnailPath(); thumbPath != "" {
		os.Remove(thumbPath)
	}
}

func handleAddForm(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func handleAddItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Parse Multipart Form
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	form, err := parseItemForm(r)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	imagePath, err := saveUploadedImage(r)
	if err != nil && err != http.ErrMissingFile {
		http.Error(w, "Error saving image", http.StatusInternalServerError)
		return
	}

	item := Item{
		Name:           form.Name,
		Description:    form.Description,
		ExpirationDate: form.ExpirationDate,
		ImagePath:      imagePath,
		ReminderDays:   form.ReminderDays,
		Tags:           form.Tags,
	}

	if _, err := CreateItem(item); err != nil {
		http.Error(w, "Error saving item", http.StatusInternalServerError)
		return
	}

	// Redirect back to form with success (simplified)
	http.Redirect(w, r, "/?success=true", http.StatusSeeOther)
}

func handleViewItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseItemID(r)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	item, err := GetItemByID(id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("templates/item.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, item)
}

func handleEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseItemID(r)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	item, err := GetItemByID(id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFiles("templates/edit.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, item)
}

func handleEditItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/list", http.StatusSeeOther)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	existing, err := GetItemByID(id)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	form, err := parseItemForm(r)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	imagePath := existing.ImagePath
	if newPath, err := saveUploadedImage(r); err == nil {
		removeItemImages(*existing)
		imagePath = newPath
	} else if err != http.ErrMissingFile {
		http.Error(w, "Error saving image", http.StatusInternalServerError)
		return
	}

	notified := existing.Notified
	if !truncateToDate(existing.ExpirationDate).Equal(truncateToDate(form.ExpirationDate)) || existing.ReminderDays != form.ReminderDays {
		notified = false
	}

	item := Item{
		ID:             id,
		Name:           form.Name,
		Description:    form.Description,
		ExpirationDate: form.ExpirationDate,
		ImagePath:      imagePath,
		ReminderDays:   form.ReminderDays,
		Tags:           form.Tags,
		Notified:       notified,
	}

	if err := UpdateItem(item); err != nil {
		http.Error(w, "Error updating item", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/item?id=%d&updated=true", id), http.StatusSeeOther)
}

type ListPageData struct {
	Items          []Item
	SortBy         string
	SortOrder      string
	ShowExpired    bool
	ShowNonExpired bool
	Search         string
}

func sortItems(items []Item, sortBy, sortOrder string) {
	ascending := sortOrder != "desc"

	sort.Slice(items, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "name":
			less = strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		case "tag":
			less = items[i].SortTag() < items[j].SortTag()
		default: // date
			less = items[i].ExpirationDate.Before(items[j].ExpirationDate)
		}
		if !ascending {
			return !less
		}
		return less
	})
}

func filterItems(items []Item, showExpired, showNonExpired bool) []Item {
	if showExpired && showNonExpired {
		return items
	}

	filtered := make([]Item, 0, len(items))
	for _, item := range items {
		if item.IsExpired() {
			if showExpired {
				filtered = append(filtered, item)
			}
		} else if showNonExpired {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func itemMatchesSearch(item Item, query string) bool {
	if strings.Contains(strings.ToLower(item.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Description), query) {
		return true
	}
	for _, tag := range item.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	if strings.Contains(item.ExpirationDate.Format("2006-01-02"), query) {
		return true
	}
	return false
}

func searchItems(items []Item, query string) []Item {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return items
	}

	filtered := make([]Item, 0, len(items))
	for _, item := range items {
		if itemMatchesSearch(item, query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func parseBoolQuery(r *http.Request, key string, defaultValue bool) bool {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return value != "0" && value != "false"
}

func (d ListPageData) listQuery(sortBy, sortOrder string) string {
	params := url.Values{}
	params.Set("sort", sortBy)
	params.Set("order", sortOrder)
	if !d.ShowExpired {
		params.Set("show_expired", "0")
	}
	if !d.ShowNonExpired {
		params.Set("show_non_expired", "0")
	}
	if d.Search != "" {
		params.Set("q", d.Search)
	}
	return params.Encode()
}

func (d ListPageData) SortLink(sortBy string) string {
	order := "asc"
	if d.SortBy == sortBy && d.SortOrder == "asc" {
		order = "desc"
	}
	return "/list?" + d.listQuery(sortBy, order)
}

func handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := GetAllItems()
	if err != nil {
		http.Error(w, "Error fetching items", http.StatusInternalServerError)
		return
	}

	sortBy := r.URL.Query().Get("sort")
	if sortBy != "name" && sortBy != "tag" {
		sortBy = "date"
	}

	sortOrder := r.URL.Query().Get("order")
	if sortOrder != "desc" {
		sortOrder = "asc"
	}

	showExpired := parseBoolQuery(r, "show_expired", true)
	showNonExpired := parseBoolQuery(r, "show_non_expired", true)
	search := strings.TrimSpace(r.URL.Query().Get("q"))

	items = filterItems(items, showExpired, showNonExpired)
	items = searchItems(items, search)
	sortItems(items, sortBy, sortOrder)

	data := ListPageData{
		Items:          items,
		SortBy:         sortBy,
		SortOrder:      sortOrder,
		ShowExpired:    showExpired,
		ShowNonExpired: showNonExpired,
		Search:         search,
	}

	tmpl, err := template.ParseFiles("templates/list.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	receivers, err := GetReceivers()
	if err != nil {
		http.Error(w, "Error fetching receivers", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/settings.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, receivers)
}

func handleAddReceiver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	email := r.FormValue("email")
	if email != "" {
		err := AddReceiver(email)
		if err != nil {
			log.Println("Error adding receiver:", err)
		}
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func handleDeleteReceiver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		err := DeleteReceiver(id)
		if err != nil {
			log.Println("Error deleting receiver:", err)
		}
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err == nil {
		err := DeleteItem(id)
		if err != nil {
			log.Println("Error deleting item:", err)
		}
	}
	http.Redirect(w, r, "/list", http.StatusSeeOther)
}

func createThumbnail(srcPath, dstPath string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Target size
	const maxW, maxH = 300, 300
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= maxW && h <= maxH {
		// No need to resize, just copy
		out, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer out.Close()
		file.Seek(0, 0)
		_, err = io.Copy(out, file)
		return err
	}

	// Calculate new dimensions maintaining aspect ratio
	newW, newH := w, h
	if w > maxW {
		newH = (h * maxW) / w
		newW = maxW
	}
	if newH > maxH {
		newW = (newW * maxH) / newH
		newH = maxH
	}

	// Simple nearest neighbor resizing (for zero dependency)
	// Actually, let's do a simple subsampling to avoid aliasing if shrinking a lot,
	// but nearest neighbor is easiest to implement without external libs.
	// For better quality without libs, we can implement a simple bilinear scaler.
	// Let's stick to a very simple subsampling/nearest neighbor for now to keep it "simple as possible".

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := x * w / newW
			srcY := y * h / newH
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if format == "png" {
		return png.Encode(out, dst)
	}
	// Default to JPEG
	return jpeg.Encode(out, dst, nil)
}

func startExpirationChecker() {
	ticker := time.NewTicker(24 * time.Hour) // Check once a day
	// Run immediately on start
	checkExpirations()

	for range ticker.C {
		checkExpirations()
	}
}

func checkExpirations() {
	log.Println("Checking for expiring items...")
	items, err := GetItemsNearExpiration() // Notify based on item's reminder_days
	if err != nil {
		log.Println("Error checking expirations:", err)
		return
	}

	for _, item := range items {
		// Send email
		receivers, err := GetReceivers()
		if err != nil {
			log.Println("Error fetching receivers:", err)
			continue
		}

		if len(receivers) == 0 {
			log.Println("No receivers configured, skipping email")
			continue
		}

		to := []string{}
		for _, r := range receivers {
			to = append(to, r.Email)
		}

		subject := fmt.Sprintf("Expiring Item: %s", item.Name)
		body := fmt.Sprintf("Your item '%s' is expiring on %s.\n\nDescription: %s\n\nView Item: http://localhost:8080/item?id=%d", item.Name, item.ExpirationDate.Format("2006-01-02"), item.Description, item.ID)

		err = SendEmail(to, subject, body)
		if err != nil {
			log.Println("Error sending email:", err)
		} else {
			log.Printf("Sent email to %v for item %s\n", to, item.Name)
			MarkAsNotified(item.ID)
		}
	}
}
