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
	"os"
	"path/filepath"
	"strconv"
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

	name := r.FormValue("name")
	description := r.FormValue("description")
	expirationStr := r.FormValue("expiration_date")
	reminderDaysStr := r.FormValue("reminder_days")

	reminderDays := 30
	if reminderDaysStr != "" {
		if rd, err := strconv.Atoi(reminderDaysStr); err == nil {
			reminderDays = rd
		}
	}

	expirationDate, err := time.Parse("2006-01-02", expirationStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Handle Image Upload
	file, handler, err := r.FormFile("image")
	var imagePath string
	if err == nil {
		defer file.Close()
		// Create uploads directory if not exists
		os.MkdirAll("uploads", os.ModePerm)

		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
		imagePath = filepath.Join("uploads", filename)
		dst, err := os.Create(imagePath)
		if err != nil {
			http.Error(w, "Error saving image", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		io.Copy(dst, file)

		// Generate Thumbnail
		thumbPath := filepath.Join("uploads", "thumb_"+filename)
		if err := createThumbnail(imagePath, thumbPath); err != nil {
			log.Println("Error creating thumbnail:", err)
			// Fallback to original image if thumbnail fails
			thumbPath = imagePath
		}
		// Store the thumbnail path in the DB for display, or logic to prefer thumbnail
		// For simplicity, let's assume we store the original path in DB,
		// but we will assume the thumbnail exists with prefix "thumb_" when rendering.
		// Actually, better to store the original path and let the template derive the thumbnail path
		// OR update the model to store both.
		// Let's stick to the plan: "Update handleAddItem to generate a thumbnail (prefix thumb_) after saving the original image."
		// The template will need to know to look for "thumb_" + filename.
		// Wait, the plan said "Update list template for grid layout".
		// I'll stick to storing the original path in the DB, and in the template/handler I'll handle the prefix.
		// Or I can just overwrite imagePath with the thumbnail path if I only want to show the thumbnail? No, I want to show the full image on details.
		// So I will just generate it here.
	}

	item := Item{
		Name:           name,
		Description:    description,
		ExpirationDate: expirationDate,
		ImagePath:      imagePath,
		ReminderDays:   reminderDays,
	}

	if err := CreateItem(item); err != nil {
		http.Error(w, "Error saving item", http.StatusInternalServerError)
		return
	}

	// Redirect back to form with success (simplified)
	http.Redirect(w, r, "/?success=true", http.StatusSeeOther)
}

func handleViewItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing item ID", http.StatusBadRequest)
		return
	}

	var id int
	_, err := fmt.Sscanf(idStr, "%d", &id)
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

func handleListItems(w http.ResponseWriter, r *http.Request) {
	items, err := GetAllItems()
	if err != nil {
		http.Error(w, "Error fetching items", http.StatusInternalServerError)
		return
	}

	funcMap := template.FuncMap{
		"now": time.Now,
	}

	tmpl, err := template.New("list.html").Funcs(funcMap).ParseFiles("templates/list.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, items)
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
