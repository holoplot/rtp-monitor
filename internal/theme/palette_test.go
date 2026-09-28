package theme

import (
	"reflect"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestPaletteComplete(t *testing.T) {
	v := reflect.ValueOf(UI)
	for i := range v.NumField() {
		if v.Field(i).IsNil() {
			t.Errorf("Palette.%s is not set", v.Type().Field(i).Name)
		}
	}
	if !reflect.DeepEqual(UI.PaneBorderFocused, lipgloss.Color(tokyoNightColors.Blue)) {
		t.Errorf("PaneBorderFocused = %v, want %s", UI.PaneBorderFocused, tokyoNightColors.Blue)
	}
}
