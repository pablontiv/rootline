package proposal

import "testing"

func TestMapValue(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"completada", "Completed"},
		{"COMPLETADA", "Completed"},
		{"activa", "In Progress"},
		{"activo", "In Progress"},
		{"pendiente", "Pending"},
		{"unknown", "unknown"},
		{"", ""},
	}
	for _, c := range cases {
		if got := mapValue(c.in); got != c.want {
			t.Errorf("mapValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
