package podcasts

import "testing"

func TestIsAffirmativeDialogResponse(t *testing.T) {
	tests := map[string]bool{
		"Yes":     true,
		"OK":      true,
		"Ok":      true,
		" yes ":   true,
		"No":      false,
		"Cancel":  false,
		"":        false,
		"Unknown": false,
	}
	for response, want := range tests {
		if got := isAffirmativeDialogResponse(response); got != want {
			t.Errorf("isAffirmativeDialogResponse(%q) = %t, want %t", response, got, want)
		}
	}
}
