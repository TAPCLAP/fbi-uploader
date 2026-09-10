package facebook

import (
	"bytes"
	"testing"
)

func TestWritePrettyJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{
			name: "empty",
			in:   nil,
			want: "",
		},
		{
			name: "whitespace",
			in:   []byte("  \n"),
			want: "",
		},
		{
			name: "pretty",
			in:   []byte(`{"data":[]}`),
			want: "{\n  \"data\": []\n}\n",
		},
		{
			name: "invalid",
			in:   []byte("{not-json"),
			want: "{not-json\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WritePrettyJSON(&buf, tt.in); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
