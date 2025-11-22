# Walkthrough - Enable Camera Input

I have enabled the camera input for the image upload field, allowing users on mobile devices to take a picture directly.

## Changes

### Frontend
- **templates/index.html**: Added `capture="environment"` to the file input field.

## Verification Results

### Automated Tests
- Ran `go build .` and it passed successfully.

### Manual Verification Steps
1. **Open on Mobile**: Access the "Add Item" page from a mobile device.
2. **Tap Image Input**: Tap the "Choose File" button for the image.
3. **Camera Option**: Verify that the option to take a photo (camera) is available and prioritized.
