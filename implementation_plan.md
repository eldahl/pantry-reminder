# Implementation Plan - Thumbnail & Grid View

## Goal Description
Generate thumbnails for uploaded images to optimize the overview page. Change the "All Items" page to a grid layout displaying these thumbnails, item names, and expiration dates.

## Proposed Changes

### Backend
#### [MODIFY] [main.go](file:///home/eldahl/Projects/pantry-reminder/main.go)
- Add `createThumbnail(srcPath, dstPath string)` function.
    - Use standard `image` package to decode.
    - Implement simple resizing (e.g., nearest neighbor or averaging) to avoid external dependencies if possible, or use `golang.org/x/image/draw` if needed. *Decision: Will use a simple box sampling implementation to keep it zero-dependency.*
- Update `handleAddItem` to generate a thumbnail (prefix `thumb_`) after saving the original image.

### Frontend
#### [MODIFY] [templates/list.html](file:///home/eldahl/Projects/pantry-reminder/templates/list.html)
- Replace the `<table>` with a grid container `<div>`.
- Each item card will show:
    - Thumbnail image (or placeholder if none).
    - Name.
    - Expiration Date.
    - Link to details.

#### [MODIFY] [static/style.css](file:///home/eldahl/Projects/pantry-reminder/static/style.css)
- Add CSS Grid styles for `.items-grid`.
- Style item cards (border, padding, shadow).

## Verification Plan

### Manual Verification
1.  **Upload Item**:
    - Add a new item with an image.
    - Check `uploads/` directory to verify `thumb_<filename>` exists.
2.  **Check Overview Page**:
    - Go to `/list`.
    - Verify items are shown in a grid.
    - Verify images are thumbnails (inspect element or visual check).
    - Verify layout is responsive.
