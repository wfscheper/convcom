package commit

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_parseHeader(t *testing.T) {
	tests := []struct {
		input     string
		want      *Commit
		wantError string
	}{
		{"(type: a description", nil, "illegal '(' character in type:1 col 0"},
		{")type: a description", nil, "illegal ')' character in type:1 col 0"},
		{"a description", nil, "illegal ' ' character in type:1 col 1"},
		{"a type: description", nil, "illegal ' ' character in type:1 col 1"},
		{"type", nil, "commit type must be followed by a colon and a single space:1 col 3"},
		{"type(scope)", nil, "commit scope must be followed by a colon and a single space:1 col 10"},
		{"type(a scope): description", nil, "illegal ' ' character in scope:1 col 6"},
		{"type(sco(pe): a description", nil, "illegal '(' character in scope:1 col 8"},
		{"type(sco)pe): a description", nil, "commit scope must be followed by a colon and a single space:1 col 9"},
		{"type(scope):  ", nil, "commit scope must be followed by a colon and a single space:1 col 13"},
		{"type(scope):  a description", nil, "commit scope must be followed by a colon and a single space:1 col 13"},
		{"type(scope): ", &Commit{Type: "type", Scope: "scope"}, ""}, // This is technically invalid, but is caught by p.Parse
		{"type(scope): description", &Commit{Type: "type", Scope: "scope", Description: "description"}, ""},
		{"type(scope):", nil, "commit scope must be followed by a colon and a single space:1 col 11"},
		{"type(scope):a description", nil, "commit scope must be followed by a colon and a single space:1 col 12"},
		{"type:  ", nil, "commit type must be followed by a colon and a single space:1 col 6"},
		{"type:  description", nil, "commit type must be followed by a colon and a single space:1 col 6"},
		{"type: ", &Commit{Type: "type"}, ""}, // This is technically invalid, but is caught by p.Parse
		{"type: description", &Commit{Type: "type", Description: "description"}, ""},
		{"type:", nil, "commit type must be followed by a colon and a single space:1 col 4"},
		{"type:description", nil, "commit type must be followed by a colon and a single space:1 col 5"},
	}
	p, err := New(&Config{})
	if err != nil {
		t.Fatal(err)
	}

	t.Parallel()
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, tt.input), func(t *testing.T) {
			commit, err := p.parseHeader(tt.input, 1)
			if err != nil && err.Error() != tt.wantError {
				t.Errorf("p.parseHeader(%s) returned an error '%v', want '%v'", tt.input, err, tt.wantError)
			} else if got, want := commit, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("p.parseHeader(%s) returned %#v, want %#v", tt.input, got, want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input     string
		want      *Commit
		wantError string
	}{
		{"type: ", nil, "commit header must contain a description"},
		{"type: description", &Commit{Type: "type", Description: "description"}, ""},
		{"type(scope): ", nil, "commit header must contain a description"},
		{"type(scope): description", &Commit{Type: "type", Scope: "scope", Description: "description"}, ""},
	}

	p, err := New(&Config{})
	if err != nil {
		t.Fatal(err)
	}

	t.Parallel()
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, tt.input), func(t *testing.T) {
			commit, err := p.Parse(tt.input)
			if err != nil && err.Error() != tt.wantError {
				t.Errorf("p.Parse(%s) returned an error '%s', want '%s'", tt.input, err, tt.wantError)
			} else if got, want := commit, tt.want; !reflect.DeepEqual(got, want) {
				t.Errorf("p.Parse(%s) returned %#v, want %#v", tt.input, got, want)
			}
		})
	}
}
