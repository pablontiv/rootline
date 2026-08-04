package extract

import "testing"

// FrontmatterBounds must confine itself to the leading "---"-delimited block.
// Thematic breaks and fenced code blocks in the body are ordinary content.
func TestFrontmatterBounds(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantOK  bool
		wantFM  string // text[start:end]
		wantRes string // text[next:]
	}{
		{
			name:    "plain frontmatter",
			text:    "---\ntitle: A\n---\n# Body\n",
			wantOK:  true,
			wantFM:  "title: A\n",
			wantRes: "# Body\n",
		},
		{
			name:    "body thematic breaks are not delimiters",
			text:    "---\ntitle: A\n---\n\nA.\n\n---\n\nB.\n\n---\n\nC.\n",
			wantOK:  true,
			wantFM:  "title: A\n",
			wantRes: "\nA.\n\n---\n\nB.\n\n---\n\nC.\n",
		},
		{
			name:    "CRLF frontmatter",
			text:    "---\r\ntitle: A\r\n---\r\nbody\r\n",
			wantOK:  true,
			wantFM:  "title: A\r\n",
			wantRes: "body\r\n",
		},
		{
			name:   "no leading delimiter",
			text:   "# Plain markdown\n\n---\n\nmore\n",
			wantOK: false,
		},
		{
			name:   "unterminated block",
			text:   "---\ntitle: A\n\nbody\n",
			wantOK: false,
		},
		{
			// The sharp case from issue #83: a "---" inside a body code fence
			// must never be mistaken for the closing delimiter.
			name:   "backtick fence hides a thematic break",
			text:   "---\ntitle: A\n\n```\n---\n```\n",
			wantOK: false,
		},
		{
			name:   "tilde fence hides a thematic break",
			text:   "---\ntitle: A\n\n~~~\n---\n~~~\n",
			wantOK: false,
		},
		{
			// A balanced fence inside a YAML block scalar must not swallow the
			// real closing delimiter that follows it.
			name:    "fence inside a block scalar still closes",
			text:    "---\nsnippet: |\n  ```\n  code\n  ```\n---\nbody\n",
			wantOK:  true,
			wantFM:  "snippet: |\n  ```\n  code\n  ```\n",
			wantRes: "body\n",
		},
		{
			// Indented fences belong to a YAML value, so an unbalanced one must
			// not open fence state and hide the closing delimiter behind it.
			name:    "unbalanced fence in a block scalar still closes",
			text:    "---\nsnippet: |\n  ```\n---\nbody\n",
			wantOK:  true,
			wantFM:  "snippet: |\n  ```\n",
			wantRes: "body\n",
		},
		{
			name:    "closing delimiter without trailing newline",
			text:    "---\ntitle: A\n---",
			wantOK:  true,
			wantFM:  "title: A\n",
			wantRes: "",
		},
		{
			name:   "opening delimiter only",
			text:   "---\n",
			wantOK: false,
		},
		{
			name:   "longer rule is not a delimiter",
			text:   "----\ntitle: A\n----\n",
			wantOK: false,
		},
		{
			name:    "empty frontmatter",
			text:    "---\n---\nbody\n",
			wantOK:  true,
			wantFM:  "",
			wantRes: "body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, next, ok := FrontmatterBounds(tt.text)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := tt.text[start:end]; got != tt.wantFM {
				t.Errorf("frontmatter = %q, want %q", got, tt.wantFM)
			}
			if got := tt.text[next:]; got != tt.wantRes {
				t.Errorf("remainder = %q, want %q", got, tt.wantRes)
			}
		})
	}
}

// Body thematic breaks must survive extraction untouched.
func TestExtract_BodyThematicBreaksPreserved(t *testing.T) {
	m := &MarkdownExtractor{}
	content := []byte("---\ntitle: Probe\n---\n\nA.\n\n---\n\nB.\n\n---\n\nC.\n")

	rec, err := m.Extract("doc.md", content)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(rec.Errors) != 0 {
		t.Fatalf("unexpected extraction errors: %v", rec.Errors)
	}
	if rec.Frontmatter["title"] != "Probe" {
		t.Errorf("title = %v, want Probe", rec.Frontmatter["title"])
	}
	if want := "A.\n\n---\n\nB.\n\n---\n\nC.\n"; rec.Body != want {
		t.Errorf("body = %q, want %q", rec.Body, want)
	}
}
