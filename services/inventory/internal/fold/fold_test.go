package fold

import "testing"

func TestFoldingNames(t *testing.T) {
	for in, want := range map[string]string{
		"(pronounced 'lĕh-'nérd 'skin-'nérd)": "pronounced leh nerd skin nerd",
		"Sgt. Pepper's Lonely Hearts":         "sgt peppers lonely hearts",
		"Guns N' Roses":                       "guns n roses",
		"AC/DC":                               "ac dc",
	} {
		if got := Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTitlesLoseTheirEditionAndTheirThe(t *testing.T) {
	for in, want := range map[string]string{
		"Toys in the Attic - 2013 Remaster":   "toys in the attic",
		"The Wall":                            "wall",
		"Presence (Remastered)":               "presence",
		"(pronounced 'lĕh-'nérd 'skin-'nérd)": "pronounced leh nerd skin nerd",
		"The":                                 "the",
	} {
		if got := Title(in); got != want {
			t.Errorf("Title(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNamesLoseTheirTheAndSpellOutAnd(t *testing.T) {
	for in, want := range map[string]string{
		"The Beatles":        "beatles",
		"Simon & Garfunkel":  "simon and garfunkel",
		"Earth, Wind & Fire": "earth wind and fire",
	} {
		if got := Name(in); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
}
