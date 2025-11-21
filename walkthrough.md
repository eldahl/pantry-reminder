# Pantry Reminder Service Walkthrough

I have successfully implemented the Pantry Reminder Service. This service allows you to track pantry items and receive email notifications when they are nearing expiration.

## Features
- **Item Registration**: Add items with a name, description, expiration date, and image.
- **Expiration Tracking**: Automatically checks for items expiring within 3 days.
- **Email Notifications**: Sends email reminders via Gmail SMTP with a link to view the item.
- **Web Interface**: Simple and clean interface for managing items and viewing details.
- **Navigation Bar**: Easy navigation between pages.
- **Overview Page**: View all registered items in a grid layout with thumbnails.
- **Thumbnails**: Automatically generates thumbnails for uploaded images.
- **Settings**: Manage email receivers for expiration notifications.

## How to Run

1. **Set Environment Variables**:
   You need to set your Gmail credentials.
   ```bash
   export GMAIL_USER="your-email@gmail.com"
   export GMAIL_PASSWORD="your-app-password"
   ```

2. **Run the Service**:
   ```bash
   go run .
   ```

3. **Access the Web Interface**:
   Open your browser and navigate to `http://localhost:8080`.

## Verification Results

### Automated Tests
I ran the automated tests to verify the database logic and expiration checking.
```bash
go test -v
```
**Output:**
```
=== RUN   TestDB
--- PASS: TestDB (0.01s)
PASS
ok      pantry-reminder 0.012s
```

### Manual Verification Steps
1. **Add an Item**:
   - Go to `http://localhost:8080`.
   - Fill in the details and upload an image.
   - Click "Add Item".
   - You should see a success message.

2. **Check for Notifications**:
   - The service checks for expirations every 24 hours.
   - On startup, it performs an immediate check.
   - If you added an item expiring within 3 days, you should receive an email.

## Project Structure
- `main.go`: Entry point, HTTP server, and background job.
- `models.go`: Database models and logic.
- `email.go`: Email sending logic.
- `templates/index.html`: Frontend form.
- `uploads/`: Directory for storing uploaded images.
- `pantry.db`: SQLite database file.
