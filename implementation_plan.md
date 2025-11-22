# Separate Image and Camera Inputs

## Goal Description
Provide two distinct buttons for adding an image: "Choose File" (standard file picker) and "Take Photo" (direct camera access).

## User Review Required
> [!NOTE]
> I will use two hidden file inputs with different names (`image_upload` and `image_camera`) and trigger them via buttons. The backend will check both fields.

## Proposed Changes

### Frontend

#### [MODIFY] [templates/index.html](file:///home/eldahl/Projects/pantry-reminder/templates/index.html)
- Remove the single `image` input.
- Add two hidden file inputs:
    - `image_upload` (accept="image/*")
    - `image_camera` (accept="image/*", capture="environment")
- Add two buttons: "Choose Image" and "Take Photo".
- Add JavaScript to trigger the respective inputs and display the selected filename.

### Backend

#### [MODIFY] [main.go](file:///home/eldahl/Projects/pantry-reminder/main.go)
- Update `handleAddItem` to check for `image_camera` first.
- If no file found in `image_camera`, check `image_upload`.

## Verification Plan

### Manual Verification
1. **Choose File**: Click "Choose Image", select a file, submit. Verify item is added with image.
2. **Take Photo**: Click "Take Photo", take a picture (or select if on desktop), submit. Verify item is added with image.
3. **Both**: (Edge case) If user does both, camera should probably take precedence or last one selected. My logic will prioritize camera if I check it first, or I can check which one is not empty.
