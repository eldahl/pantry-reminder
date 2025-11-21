# Implementation Plan - Email Receivers Settings

## Goal Description
Allow users to manage a list of email addresses that will receive expiration notifications. This will be done via a new "Settings" page.

## Proposed Changes

### Backend
#### [MODIFY] [models.go](file:///home/eldahl/Projects/pantry-reminder/models.go)
- Add `Receiver` struct (`ID`, `Email`).
- Update `InitDB` to create `receivers` table.
- Add `AddReceiver(email string) error`.
- Add `GetReceivers() ([]Receiver, error)`.
- Add `DeleteReceiver(id int) error`.

#### [MODIFY] [main.go](file:///home/eldahl/Projects/pantry-reminder/main.go)
- Add route `GET /settings`.
- Add route `POST /settings/add-receiver`.
- Add route `POST /settings/delete-receiver`.
- Update `checkExpirations` to fetch receivers from DB and send emails to all of them.

### Frontend
#### [NEW] [templates/settings.html](file:///home/eldahl/Projects/pantry-reminder/templates/settings.html)
- List current receivers with a "Delete" button.
- Form to add a new receiver.

#### [MODIFY] [templates/index.html](file:///home/eldahl/Projects/pantry-reminder/templates/index.html)
- Update Nav Bar to include "Settings".

#### [MODIFY] [templates/item.html](file:///home/eldahl/Projects/pantry-reminder/templates/item.html)
- Update Nav Bar to include "Settings".

#### [MODIFY] [templates/list.html](file:///home/eldahl/Projects/pantry-reminder/templates/list.html)
- Update Nav Bar to include "Settings".

## Verification Plan

### Manual Verification
1.  **Settings Page**:
    - Go to `/settings`.
    - Add an email. Verify it appears in the list.
    - Delete an email. Verify it disappears.
2.  **Email Sending**:
    - Add a valid email (e.g., your own if testing, or just verify log output).
    - Trigger expiration check (restart server or wait).
    - Verify logs show "Sent email to [email]".
