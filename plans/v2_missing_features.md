# LazyHue V2 Missing Features Analysis

**Date:** 2026-01-13
**Status:** Analysis Complete
**Purpose:** Document features present in V1 but missing or incomplete in V2

## Overview

This document compares the V1 UI implementation (`internal/app/`, `internal/ui/`) with the V2 UI implementation (`internal/app2/`, `internal/ui2/`) to identify functionality gaps that need to be addressed during the migration.

---

## High Priority Missing Features

### 1. Bridge Pairing UI
**Status:** ✗ Missing
**V1 Reference:** `internal/ui/panels/pairing.go`

**Description:**
- V1 has a full pairing panel with visual progress indicator
- Timeout countdown with color transitions (green→yellow→red)
- Status messages during pairing process
- V2 has no pairing UI - connection happens silently

**Implementation Notes:**
- Need to create pairing panel in V2
- Show "Press button on bridge" message
- Display countdown timer
- Visual feedback for success/failure

---

### 2. Create Room
**Status:** ✗ Missing
**V1 Reference:** `internal/app/handlers.go:1115-1193`

**Description:**
- Form for creating new rooms with archetype selection
- Device membership selection during creation
- Validation and error handling

**V2 Files to Modify:**
- `internal/app2/app.go` - Add key handler for 'n'
- `internal/app2/rendering.go` - Build create room form
- Need to call `hue.Bridge.CreateRoom()` API

**Required Form Fields:**
- Room name (text input)
- Room archetype (dropdown)
- Device selection (multi-select or checkboxes)

---

### 3. Create Zone
**Status:** ✗ Missing
**V1 Reference:** `internal/app/handlers.go:1196-1274`

**Description:**
- Form for creating new zones with archetype selection
- Device membership selection during creation
- Validation and error handling

**V2 Files to Modify:**
- `internal/app2/app.go` - Add key handler for 'N'
- `internal/app2/rendering.go` - Build create zone form
- Need to call `hue.Bridge.CreateZone()` API

**Required Form Fields:**
- Zone name (text input)
- Zone archetype (dropdown)
- Device selection (multi-select or checkboxes)

---

### 4. Delete Entity
**Status:** ✗ Missing
**V1 Reference:** `internal/app/handlers.go:1276-1335`

**Description:**
- Confirmation dialog before deleting rooms, zones, scenes, lights
- Prevents accidental deletion
- Shows entity name and type in confirmation

**V2 Files to Modify:**
- `internal/app2/app.go` - Add key handler for 'd' or Delete key
- Need confirmation popup/dialog
- Call appropriate delete API based on entity type

**Implementation:**
- Show confirmation: "Delete [entity type] '[name]'? (y/n)"
- Validate that deletion is allowed (can't delete certain entities)
- Handle errors gracefully

---

### 5. Rename Entity (Completion)
**Status:** ~ Partial
**V1 Reference:** `internal/app/handlers.go:828-865`

**Description:**
- V2 has rename mode started but not fully connected
- Text input for new name exists
- Missing: Form submission and API call integration

**V2 Files to Modify:**
- `internal/app2/app.go` - Complete rename handler
- Connect rename form submission to API calls
- Handle different entity types (lights, rooms, zones, devices, scenes)

**Current State:**
- Rename mode activates with 'r' key
- Text input appears
- Missing: Actual rename API call on Enter

---

## Medium Priority Missing Features

### 6. Edit Device Settings
**Status:** ✗ Missing
**V1 Reference:** `internal/app/handlers.go:504-601`

**Description:**
- Motion sensor configuration (sensitivity)
- Device-specific settings
- Product archetype editing

**V2 Files to Modify:**
- `internal/app2/app.go` - Add handler for 'e' key on device selection
- `internal/app2/rendering.go` - Build device edit form

**Required Features:**
- Motion sensor sensitivity slider
- Archetype dropdown (already implemented for lights)
- Device name editing

---

### 7. Edit Room/Zone Properties
**Status:** ~ Partial
**V1 Reference:**
- `internal/app/handlers.go:1338-1484` (rooms)
- `internal/app/handlers.go:1486-1632` (zones)

**Description:**
- Edit room/zone archetype
- Manage device membership (add/remove devices)
- Visual indication of current members

**V2 Files to Modify:**
- `internal/app2/rendering.go` - Add room/zone edit forms
- Need multi-select or checkbox list for device membership

**Required Features:**
- Archetype dropdown
- Device membership management
- Save/cancel buttons

---

### 8. Room/Zone Member Management
**Status:** ✗ Missing
**V1 Reference:** `internal/app/handlers.go:1376-1419`

**Description:**
- Show all devices with toggles to add/remove from room/zone
- Visual indication of which devices are in which room
- Bulk device membership management

**Implementation Notes:**
- Part of the edit room/zone feature
- Could also be standalone interface
- Useful for reorganizing spaces

---

### 9. Flat View Tabs
**Status:** ✗ Missing
**V1 Reference:** `internal/ui/panels/home_tabbed.go`

**Description:**
- V1 has separate tabs showing:
  - "Home" - Hierarchical tree (like V2 currently has)
  - "Lights" - Flat list of all lights
  - "Devices" - Flat list of all devices
  - "Scenes" - Flat list of all scenes
- V2 only has the hierarchical tree view

**V2 Files to Modify:**
- `internal/ui2/panels/tree.go` - Add tab support
- `internal/app2/app.go` - Handle tab switching

**Benefits:**
- Easier to find specific lights without navigating tree
- Better for managing many entities
- Alternative organization method

---

### 10. Activity Log Polish
**Status:** ~ Partial
**V1 Reference:** `internal/ui/panels/logpanel.go`

**Description:**
- V2 has basic activity log but missing some features:
  - Event type filtering
  - Colored indicators per event type
  - Rich resource-specific details
  - Toggle visibility with 'a' key

**Current V2 State:**
- Has log viewport
- Shows events with timestamps
- Shows bridge name (recently added)
- Missing: Toggle key, filtering, enhanced formatting

**V2 Files to Modify:**
- `internal/app2/app.go` - Add 'a' key handler for toggle
- `internal/app2/activity.go` - Enhanced rendering options
- Add filtering/search capability

---

## Low Priority Missing Features

### 11. Entertainment Area Details
**Status:** ✗ Missing
**V1 Reference:** `internal/ui/panels/details_entertainment.go`

**Description:**
- View entertainment configuration details
- Shows channels, lights, positions
- Stream proxy information

**Notes:**
- Entertainment areas are less commonly used
- Can be added when user requests it
- Low priority for general usage

---

### 12. Live Form Updates
**Status:** ~ Partial
**V1 Reference:** `internal/ui/panels/popup.go:183-189`

**Description:**
- V1 had `ShowFormLive()` for real-time updates
- Changes applied immediately as user adjusts sliders
- Useful for light brightness/color adjustments

**Current V2 State:**
- Forms exist and work well
- Updates happen on form submission, not during editing
- Could add live preview for better UX

**Implementation:**
- Add option to send updates on every field change
- Debounce rapid changes
- Visual feedback that live mode is active

---

## Missing Keyboard Shortcuts

**V1 Keybindings Missing in V2:**

| Key | V1 Action | V2 Status | Priority |
|-----|-----------|-----------|----------|
| `n` | Create Room | ✗ Not bound | High |
| `N` | Create Zone | ✗ Not bound | High |
| `d` | Delete Entity | ✗ Not bound | High |
| `e` | Edit Device | ✗ Not bound | Medium |
| `a` | Toggle Activity Log | ✗ Not bound | Medium |

**V1 Reference:** `internal/app/bindings.go`

**V2 Files to Modify:**
- `internal/app2/app.go` - Add key handlers in Update function
- Follow existing pattern for handling keys

---

## Summary Table

| Feature | V1 Status | V2 Status | Priority | Files to Reference |
|---------|-----------|-----------|----------|-------------------|
| Bridge Pairing UI | ✓ Full | ✗ Missing | High | ui/panels/pairing.go |
| Create Room | ✓ Full | ✗ Missing | High | app/handlers.go:1115 |
| Create Zone | ✓ Full | ✗ Missing | High | app/handlers.go:1196 |
| Delete Entity | ✓ Full | ✗ Missing | High | app/handlers.go:1276 |
| Rename Entity | ✓ Full | ~ Partial | High | app/handlers.go:828 |
| Edit Device Settings | ✓ Full | ✗ Missing | Medium | app/handlers.go:504 |
| Edit Room Properties | ✓ Full | ~ Partial | Medium | app/handlers.go:1338 |
| Edit Zone Properties | ✓ Full | ~ Partial | Medium | app/handlers.go:1486 |
| Room Member Mgmt | ✓ Full | ✗ Missing | Medium | app/handlers.go:1376 |
| Flat View Tabs | ✓ Full | ✗ Missing | Medium | ui/panels/home_tabbed.go |
| Activity Log Polish | ✓ Full | ~ Partial | Medium | ui/panels/logpanel.go |
| Entertainment Details | ✓ Full | ✗ Missing | Low | ui/panels/details_entertainment.go |
| Live Form Updates | ✓ Full | ~ Partial | Low | ui/panels/popup.go:183 |

**Legend:**
- ✓ Full: Fully implemented
- ~ Partial: Started but incomplete
- ✗ Missing: Not implemented

---

## Implementation Recommendations

### Phase 1: Core CRUD Operations (High Priority)
1. **Complete Rename** - Finish connecting the existing rename UI to API
2. **Add Delete** - Implement delete confirmation and API calls
3. **Add Create Room/Zone** - Forms and API integration
4. **Bridge Pairing UI** - Visual feedback for pairing process

**Estimated Effort:** 2-3 days
**Impact:** Enables full entity management in V2

---

### Phase 2: Enhanced Editing (Medium Priority)
1. **Edit Device Settings** - Motion sensor config, archetype changes
2. **Room/Zone Property Editing** - Archetype and member management
3. **Keyboard Shortcuts** - Bind n, N, d, e, a keys
4. **Activity Log Toggle** - Add 'a' key to show/hide log

**Estimated Effort:** 2 days
**Impact:** Improves workflow and accessibility

---

### Phase 3: Alternative Views (Medium Priority)
1. **Flat View Tabs** - Add Lights, Devices, Scenes tabs
2. **Activity Log Enhancements** - Filtering, better formatting

**Estimated Effort:** 2 days
**Impact:** Better for users with many entities

---

### Phase 4: Polish & Edge Cases (Low Priority)
1. **Entertainment Areas** - View configuration details
2. **Live Form Updates** - Real-time preview during editing

**Estimated Effort:** 1-2 days
**Impact:** Nice-to-have features

---

## File Reference Guide

### V1 Files to Study for Implementation

**Action Handlers:**
- `internal/app/handlers.go` - All CRUD operations
- `internal/app/dispatch.go` - Action dispatch logic
- `internal/app/bindings.go` - Keyboard shortcuts

**UI Components:**
- `internal/ui/panels/popup.go` - Form UI and interactions
- `internal/ui/panels/detailspanel.go` - Detail rendering
- `internal/ui/panels/logpanel.go` - Activity logging
- `internal/ui/panels/pairing.go` - Bridge pairing UI
- `internal/ui/panels/home_tabbed.go` - Tabbed panel layout
- `internal/ui/panels/details_*.go` - Entity-specific detail views

### V2 Files to Modify

**Main Application:**
- `internal/app2/app.go` - Main Update handler, key bindings
- `internal/app2/rendering.go` - Detail rendering, form building
- `internal/app2/activity.go` - Event handling and logging

**UI Components:**
- `internal/ui2/components/form.go` - Form implementation
- `internal/ui2/panels/tree.go` - Tree panel (could add tabs)
- `internal/ui2/layout/layout.go` - Layout system

**Service Layer (No changes needed):**
- `internal/hue/actions.go` - Already has all CRUD APIs
- `internal/hue/bridge.go` - Bridge management
- `internal/hue/state.go` - State management

---

## Notes

- The service layer (`internal/hue/`) is complete and unchanged between V1 and V2
- All APIs needed for missing features already exist
- Focus is on UI/UX implementation in app2/ and ui2/
- Many features are partially implemented (rename, edit) and just need completion
- The migration to Bubble Tea V2 is nearly complete from an architecture standpoint
- Most missing features are about exposing existing APIs through the UI

---

## Related Documents

- `charm_v2_migration.md` - Overall migration plan
- `form_components_analysis.md` - Form field types and capabilities
- `details_refactor.md` - Detail panel refactoring notes
- `lazyhue_technical_plan.md` - Original technical architecture
