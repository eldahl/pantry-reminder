# Add Configurable Reminder Days

## Goal Description
Allow users to configure how many days before expiration they should be reminded for each item. The default value will be 30 days.

## User Review Required
> [!NOTE]
> Existing items will need to have a default value set for the new `reminder_days` column. I will set this to 30 days during the migration.

## Proposed Changes

### Backend

#### [MODIFY] [models.go](file:///home/eldahl/Projects/pantry-reminder/models.go)
- Update `Item` struct to include `ReminderDays int`.
- Update `InitDB` to add `reminder_days` column to `items` table if it doesn't exist.
    - Use `ALTER TABLE items ADD COLUMN reminder_days INTEGER DEFAULT 30;` inside a check or try-catch block (or just execute and ignore error if column exists, or check schema).
- Update `CreateItem` to insert `reminder_days`.
- Update `GetItemsNearExpiration` to use the item's `reminder_days` for the check instead of a fixed parameter.
    - Query change: `expiration_date <= DATE('now', '+' || reminder_days || ' days')` (SQLite).
- Update `GetItemByID` and `GetAllItems` to scan `reminder_days`.

#### [MODIFY] [main.go](file:///home/eldahl/Projects/pantry-reminder/main.go)
- Update `handleAddItem` to parse `reminder_days` from the form. Default to 30 if empty/invalid.
- Update `checkExpirations` to call `GetItemsNearExpiration` without arguments (or ignore the arg).

### Frontend

#### [MODIFY] [templates/index.html](file:///home/eldahl/Projects/pantry-reminder/templates/index.html)
- Add an input field for "Reminder Days" (type number, default 30).

#### [MODIFY] [templates/item.html](file:///home/eldahl/Projects/pantry-reminder/templates/item.html)
- Display the "Reminder Days" value in the details view.

## Verification Plan

### Automated Tests
- None.

### Manual Verification
1. **Migration**: Run the app and ensure no errors on startup (DB migration).
2. **Add Item**: Add a new item with a custom reminder period (e.g., 5 days).
3. **View Item**: Check the item details page to see if "Reminder Days: 5" is displayed.
4. **Default Value**: Add an item without changing the reminder days and verify it saves as 30.
5. **Notification Logic**: (Hard to test in real-time without mocking time or DB) - I will verify the SQL query logic by review or by temporarily setting a very long reminder period for an item expiring soon and seeing if it gets picked up (or vice versa).
