# SelectList Dialog Example

This example demonstrates the `SelectListModel` component with both single-select and multi-select modes.

## Features Demonstrated

- **Single-select mode**: Choose one option using radio buttons `(•)` / `( )`
- **Multi-select mode**: Choose multiple options using checkboxes `[x]` / `[ ]`
- **Dialog flow**: Proper opening, interaction, and closing of dialogs
- **Result handling**: Display of selected options and cancellation

## How to Run

```bash
go run ui/example/selectlist/main.go
```

## Controls

### Main Menu
- `s` - Open single-select dialog
- `m` - Open multi-select dialog
- `q` or `Ctrl+C` - Quit application

### Dialog Controls
- `↑/↓` - Navigate through options
- `Space` - Toggle selection (multi-select) or select item (single-select)
- `Tab` - Switch focus between list and OK button
- `Enter` - Confirm selection
- `Esc` - Cancel dialog

## Example Usage

1. Run the example
2. Press `s` to see single-select mode with radio buttons
3. Press `m` to see multi-select mode with checkboxes
4. Use the dialog controls to make selections
5. Press `Enter` to confirm or `Esc` to cancel
6. View the results displayed on the main screen

## Code Structure

The example shows how to:
- Create `SelectListModel` instances with different options
- Handle `SelectListCloseMsg` messages
- Manage dialog state and focus
- Display results to the user
