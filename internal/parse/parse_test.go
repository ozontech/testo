package parse

import "testing"

func TestBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "simple value",
			s:    "true",
			want: true,
		},
		{
			name: "padded value",
			s:    "   1 ",
			want: true,
		},
		{
			name: "empty value",
			s:    "",
			want: false,
		},
		{
			name: "padded false value",
			s:    "       f ",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Bool(tt.s)

			if tt.want != got {
				t.Errorf("Bool() = %v, want %v", got, tt.want)
			}
		})
	}
}
