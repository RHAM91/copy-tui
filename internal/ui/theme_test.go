package ui

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// useTheme applies the theme with id for one test and restores the default
// afterwards, since the styles are shared by the whole package.
func useTheme(t *testing.T, id string) {
	t.Helper()
	th, ok := ThemeByID(id)
	if !ok {
		t.Fatalf("theme %q not found", id)
	}
	applyTheme(th)
	t.Cleanup(func() { applyTheme(themes[0]) })
}

func TestThemesListTheDefaultFirstThenTheNewOnes(t *testing.T) {
	var got [][2]string
	for _, th := range Themes() {
		got = append(got, [2]string{th.ID, th.Name})
	}
	want := [][2]string{
		{"catppuccin-mocha", "Catppuccin Mocha"},
		{"kanagawa-wave", "Kanagawa Wave"},
		{"kanagawa-dragon", "Kanagawa Dragon"},
		{"tokyo-night", "Tokyo Night"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Themes() = %v, want %v", got, want)
	}
	if DefaultThemeID != "catppuccin-mocha" {
		t.Fatalf("DefaultThemeID = %q", DefaultThemeID)
	}
	if _, ok := ThemeByID("solarized"); ok {
		t.Fatal("unknown theme id should not be found")
	}
}

func TestApplyThemeRecolorsTheStyles(t *testing.T) {
	for _, th := range Themes() {
		t.Run(th.ID, func(t *testing.T) {
			useTheme(t, th.ID)
			checks := []struct {
				name string
				got  any
				want lipgloss.Color
			}{
				{"active border", stylePanelActive.GetBorderTopForeground(), th.Accent},
				{"directory", styleDir.GetForeground(), th.Dir},
				{"cursor", styleCursor.GetBackground(), th.Surface0},
				{"error", styleStatusError.GetForeground(), th.Error},
				{"dialog border", styleDialog.GetBorderTopForeground(), th.Dialog},
				{"danger dialog border", styleDialogDanger.GetBorderTopForeground(), th.Error},
				{"active option", styleOptionActive.GetBackground(), th.Dialog},
			}
			for _, c := range checks {
				if c.got != c.want {
					t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
				}
			}
		})
	}
}

func TestEveryThemeSetsEveryColor(t *testing.T) {
	for _, th := range Themes() {
		v := reflect.ValueOf(th)
		for i := range v.NumField() {
			f := v.Field(i)
			if c, ok := f.Interface().(lipgloss.Color); ok && c == "" {
				t.Errorf("%s: %s is empty", th.ID, v.Type().Field(i).Name)
			}
		}
	}
}
