package ui

import "testing"

// The layer that owns the keyboard and the overlay is decided in one place;
// these cases pin its precedence, from the topmost layer down.
func TestActiveModalPrecedence(t *testing.T) {
	tests := []struct {
		name string
		m    Model
		want modalKind
	}{
		{"nothing open", Model{}, modalNone},
		{"search prompt", Model{search: search{active: true}}, modalSearch},
		{"help over search", Model{helpOpen: true, search: search{active: true}}, modalHelp},
		{"bookmarks over help", Model{bookmarks: &bookmarkList{}, helpOpen: true}, modalBookmarks},
		{"input over bookmarks", Model{input: &inputPrompt{}, bookmarks: &bookmarkList{}}, modalInput},
		{"confirm over bookmarks", Model{confirm: &confirmPrompt{}, bookmarks: &bookmarkList{}}, modalConfirm},
		{"theme picker over help", Model{themes: &themePicker{}, helpOpen: true}, modalTheme},
		{"sync over help", Model{sync: &syncPrompt{}, helpOpen: true}, modalSync},
		{"input over help", Model{input: &inputPrompt{}, helpOpen: true}, modalInput},
		{"confirm over input", Model{confirm: &confirmPrompt{}, input: &inputPrompt{}}, modalConfirm},
		{"operation over confirm", Model{op: &operation{}, confirm: &confirmPrompt{}}, modalOp},
		{"conflict over operation", Model{op: &operation{conflict: &conflictPrompt{}}}, modalConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.activeModal(); got != tt.want {
				t.Fatalf("activeModal() = %v, want %v", got, tt.want)
			}
		})
	}
}
