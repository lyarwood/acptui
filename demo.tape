Output demo.gif
Require ./bin/acptui
Set Shell bash
Set Width 1400
Set Height 700
Set FontSize 14
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# The mock API server must be running on :9999 before this tape.
# The Makefile handles starting/stopping it.

# --- Project Picker ---
Type "AMBIENT_CONFIG=/tmp/acptui-demo/config.json ./bin/acptui"
Enter
Sleep 2s

# Navigate projects
Down
Sleep 500ms
Down
Sleep 500ms

# Select a project
Up
Sleep 300ms
Up
Sleep 300ms
Enter
Sleep 2s

# --- Session List ---
# Navigate sessions
Down
Sleep 500ms
Down
Sleep 500ms

# Toggle help
Type "?"
Sleep 2s
Type "?"
Sleep 500ms

# View session details
Type "d"
Sleep 3s
Escape
Sleep 1s

# Filter by phase
Type "/"
Sleep 500ms
Type "phase:Running"
Enter
Sleep 2s

# Clear filter
Type "/"
Sleep 300ms
Ctrl+U
Enter
Sleep 1s

# --- Chat View ---
# Select first session to open chat
Up
Sleep 300ms
Up
Sleep 300ms
Up
Sleep 300ms
Up
Sleep 300ms
Enter
Sleep 3s

# Toggle thinking block
Tab
Sleep 2s
Tab
Sleep 1s

# Open tasks viewer
Ctrl+T
Sleep 2s
Escape
Sleep 1s

# Open file browser
Ctrl+F
Sleep 2s
Escape
Sleep 1s

# Back to session list
Escape
Sleep 1s

# --- Create Session ---
Type "n"
Sleep 1s

# Type prompt
Type "Add rate limiting to the /api/v2 endpoints"
Sleep 500ms

# Tab to repo field
Tab
Sleep 500ms

# Use repo suggestion
Right
Sleep 500ms

# Tab to model
Tab
Sleep 300ms
Tab
Sleep 500ms

# Cycle model
Right
Sleep 500ms
Right
Sleep 500ms

# Tab to workflow
Tab
Sleep 500ms
Right
Sleep 500ms
Right
Sleep 1s

# Cancel
Escape
Sleep 1s

# Back to project list
Escape
Sleep 1s

# Quit
Type "q"
Sleep 1s
