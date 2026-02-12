package ai

import "testing"

func TestStripTrailingCharCount(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Главная неудача похудения (168 символов).", "Главная неудача похудения"},
		{"Главная неудача похудения (168 символов)", "Главная неудача похудения"},
		{"Some text (168 characters).", "Some text"},
		{"Some text (168 character)", "Some text"},
		{"No suffix here.", "No suffix here."},
		{"   Text with spaces (12 символов).   ", "Text with spaces"},
	}

	for _, c := range cases {
		if got := StripTrailingCharCount(c.in); got != c.want {
			t.Fatalf("StripTrailingCharCount(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestApproxWordCountFromHTML(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"<p>Hello world</p>", 2},
		{"<h2>Title</h2><p>Hello&nbsp;world</p>", 3},
		{"<p>Привет мир</p>", 2},
		{"<p>one two</p><ul><li>three</li><li>four</li></ul>", 4},
	}

	for _, c := range cases {
		if got := ApproxWordCountFromHTML(c.in); got != c.want {
			t.Fatalf("ApproxWordCountFromHTML(%q) = %d; want %d", c.in, got, c.want)
		}
	}
}
