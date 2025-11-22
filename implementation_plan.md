# Enable Camera Input

## Goal Description
Allow users to directly use their camera to take a picture of the item when adding it.

## User Review Required
> [!NOTE]
> This feature primarily affects mobile devices. Desktop browsers may ignore the `capture` attribute.

## Proposed Changes

### Frontend

#### [MODIFY] [templates/index.html](file:///home/eldahl/Projects/pantry-reminder/templates/index.html)
- Update the file input to include `capture="environment"`.

## Verification Plan

### Manual Verification
- Open the "Add Item" page on a mobile device (or simulate mobile view if possible, though camera access might be limited in simulation).
- Click "Choose File".
- Verify that the camera option is presented directly or as a primary option.
