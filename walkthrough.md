# Walkthrough - Configurable Reminder Days

I have added a configurable `reminder_days` field to items, allowing users to specify how many days before expiration they should be reminded. The default is 30 days.

## Changes

### Backend
- **models.go**:
    - Updated `Item` struct to include `ReminderDays`.
    - Updated `InitDB` to add the `reminder_days` column to the `items` table (migration).
    - Updated `CreateItem`, `GetItemByID`, `GetAllItems` to handle the new field.
    - Updated `GetItemsNearExpiration` to use the item's `reminder_days` in the SQL query instead of a fixed value.
- **main.go**:
    - Updated `handleAddItem` to parse `reminder_days` from the form.
    - Updated `checkExpirations` to call `GetItemsNearExpiration` without arguments.

### Frontend
- **templates/index.html**: Added a "Reminder Days" input field (default 30) to the add item form.
- **templates/item.html**: Displayed the "Reminder Days" value on the item detail page.

## Verification Results

### Automated Tests
- Ran `go test .` and it passed successfully.

### Manual Verification Steps
1. **Add Item**:
    - Go to the "Add Item" page.
    - Enter item details and set "Reminder Days" to a custom value (e.g., 7).
    - Submit the form.
2. **View Item**:
    - Click on the newly added item.
    - Verify that "Reminder Days: 7 days before" is displayed.
3. **Default Value**:
    - Add another item but leave "Reminder Days" as 30.
    - Verify it saves as 30.
