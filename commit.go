// Copyright 2020 Walter Scheper
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commit

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Commit represents a conventional commit.
type Commit struct {
	// Type is a noun that describes what type of commit this is.
	Type string
	// Scope is an optional noun describing the section of code affected by the commit.
	Scope string
	// Description is a short summary of the changes in the commit.
	Description string
	// Header is the full header of the commit message.
	Header string
	// MergeHeader is merge header if any.
	MergeHeader string
	// Body is a free-from long description of the changes in the commit.
	Body string
	// Footers are token/value pairs at the end of a commit.
	Footers []Footer
	// Mentions are people or groups explicitly mentioned in the commit.
	Mentions []string
	// References are ticket IDs or other resources refereneced by the commit.
	References References
	// Notes is a list of notes in the commit.
	Notes Notes
	// Reverts is what this commit reverts
	Reverts Reverts
	// IsBreaking is a boolean indicating if this is an API breaking change
	IsBreaking bool
}

// Config reprensts the configuration for how to parse a commit.
type Config struct {
	// MergePattern is a regex that matches a merge header.
	MergePattern string
	mergePattern *regexp.Regexp
	// MergeGroups is a list of capturing groups for MergePattern.
	//
	// The order of the names should match the order of the groups in
	// MergePattern.
	MergeGroups []string
	// ReferenceActions are the keywords used to reference an issue. These matched
	// case insensitive.
	//
	// Default: []string{"close", "closes", "closed", "fix", "fixes", "fixed", "resolve", "resolves", "resolved"}
	ReferenceActions []string
	// IssuePrefixes is a list of prefixes that start an issue. Eg. In `gh-123`, `gh-` is the prefix.
	//
	// Default: []string{"#"}
	IssuePrefixes []string
	// IssuePrefixesCaseSensitive is a boolean indicating if the prefixes in
	// IssuePrefixes should be considered case sensitive.
	//
	// Default: false
	IssuePrefixesCaseSensitive bool
	// NoteKeywords is a list of keywords that mark important notes. This value is case insensitive.
	//
	// Default: []string{"BREAKING CHANGE"}
	NoteKeywords []string
	// FieldPattern is a regular expression that matches other fields.
	//
	// Default: "^-(.*?)-$"
	FieldPattern string
	fieldPattern *regexp.Regexp
	// RevertPattern is a regular expression to match what a commit reverts.
	//
	// Default: "^Revert\s"([\s\S]*)"\s*This reverts commit (\w*)\."
	RevertPattern string
	revertPattern *regexp.Regexp
	// RevertGroups defines which field the capturing groups of RevertPattern
	// capture. The order of the slice should correspond to the order of
	// RevertPattern's capturing groups.
	//
	// Default: []string{"header", "hash"}
	RevertGroups []string
	// CommentCharacter sets what the comment character is. If empty no comments are stripped.
	CommentCharacter string
	// ErrorCallback is a function to call when a commit cannot be parsed. If nil,
	// then the parser will return an error.
	ErrorCallback func(message string, line, char int) error
}

var (
	referenceActions = []string{"close", "closes", "closed", "fix", "fixes", "fixed", "resolve", "resolves", "resolved"}
	issuePrefixes    = []string{"#"}
	noteKeywords     = []string{"BREAKING CHANGE"}
	fieldPattern     = regexp.MustCompile(`^-(.*?)-$`)
	revertPattern    = regexp.MustCompile(`^Revert\s"([\s\S]*)"\s*This reverts commit (\w*)\.`)
	revertGropus     = []string{"header", "hash"}
)

// Parser is a commit parser
type Parser struct {
	cfg *Config
}

// New returns a new Parser
func New(cfg *Config) (p *Parser, err error) {
	if "" != cfg.FieldPattern {
		cfg.fieldPattern, err = regexp.Compile(cfg.FieldPattern)
		if err != nil {
			return nil, fmt.Errorf("cannot parse FieldPattern /%s/: %w", cfg.FieldPattern, err)
		} else {
			cfg.fieldPattern = fieldPattern
		}
	}
	if nil == cfg.IssuePrefixes {
		cfg.IssuePrefixes = issuePrefixes
	}
	if "" != cfg.MergePattern {
		cfg.mergePattern, err = regexp.Compile(cfg.MergePattern)
		if err != nil {
			return nil, fmt.Errorf("cannot parse MergePattern /%s/: %w", cfg.MergePattern, err)
		}
	}
	if nil == cfg.NoteKeywords {
		cfg.NoteKeywords = noteKeywords
	}
	if nil == cfg.ReferenceActions {
		cfg.ReferenceActions = referenceActions
	}
	if nil == cfg.RevertGroups {
		cfg.RevertGroups = revertGropus
	}
	if "" != cfg.RevertPattern {
		cfg.revertPattern, err = regexp.Compile(cfg.RevertPattern)
		if err != nil {
			return nil, fmt.Errorf("cannot parse RevertPattern /%s/: %w", cfg.RevertPattern, err)
		}
	} else {
		cfg.revertPattern = revertPattern
	}
	return &Parser{cfg: cfg}, nil
}

// Parse parses a commit message and returns a Commit.
//
// If the commit message does not match the specification,
// then an empty Commit will be returned
// along with an error describing the parse error.
func (p *Parser) Parse(s string) (c *Commit, err error) {
	// split message into lines
	lines := strings.Split(s, "\n")
	header := lines[0]

	// parse header
	c, err = p.parseHeader(header, 1)
	if err != nil {
		return nil, err
	}
	switch {
	case "" == c.Type:
		return nil, errors.New("commit header must contain a type")
	case "" == c.Description:
		return nil, errors.New("commit header must contain a description")
	}
	return c, nil
}

func (p *Parser) parseHeader(h string, line int) (*Commit, error) {
	var inDescription, inScope bool
	var b strings.Builder
	commit := &Commit{}
	for i, c := range h {
		if inDescription {
			// part of the description so write to buffer
			if _, err := b.WriteRune(c); err != nil {
				return nil, fmt.Errorf("could not write character '%c' to internal buffer: %w", c, err)
			}
			continue
		}
		// still parsing the type/description
		switch c {
		case '(':
			if inScope {
				return nil, ParseError{
					Char:    i,
					Line:    line,
					Message: "illegal '(' character in scope",
				}
			}
			inScope = true
			commit.Type = b.String()
			if "" == commit.Type {
				return nil, ParseError{
					Char:    i,
					Line:    line,
					Message: "illegal '(' character in type",
				}
			}
			b.Reset()
		case ')':
			// if scope hasn't started this is an illegal character
			if !inScope {
				return nil, ParseError{
					Char:    i,
					Line:    line,
					Message: "illegal ')' character in type",
				}
			}
			// done with scope
			commit.Scope = b.String()
			b.Reset()
		case ':':
			// peek ahead, the : must be followed by exactly one space and a description
			var char int
			var length = len(h)
			switch {
			case i+1 == length:
				// : is the last character in the string
				char = i
			case ' ' != h[i+1]:
				char = i + 1
			case i+2 < length && ' ' == h[i+2]:
				// no description is technically an error, but that will be handled by the caller
				char = i + 2
			}
			if 0 != char {
				return nil, ParseError{
					Char:    char,
					Line:    line,
					Message: fmtError("commit %s must be followed by a colon and a single space", inScope),
				}
			}
			if !inScope {
				// no scope, just type
				commit.Type = b.String()
				b.Reset()
			}
			inDescription = true
		case ' ':
			err := ParseError{
				Char:    i,
				Line:    line,
				Message: fmtError("illegal ' ' character in %s", inScope),
			}
			return nil, err
		default:
			if "" != commit.Scope {
				// we finished scope but didn't hit a ':' above
				return nil, ParseError{
					Char:    i,
					Line:    line,
					Message: "commit scope must be followed by a colon and a single space",
				}
			}
			if _, err := b.WriteRune(c); err != nil {
				return nil, fmt.Errorf("could not write character '%c' to internal buffer: %w", c, err)
			}
		}
	}
	if !inDescription {
		// never entered the description
		return nil, ParseError{
			Char:    len(h) - 1,
			Line:    line,
			Message: fmtError("commit %s must be followed by a colon and a single space", inScope),
		}
	}
	commit.Description = strings.TrimSpace(b.String())
	return commit, nil
}

func fmtError(format string, inScope bool) string {
	field := "type"
	if inScope {
		field = "scope"
	}
	return fmt.Sprintf(format, field)
}

type ParseError struct {
	Line, Char int
	Message    string
}

func (p ParseError) Error() string {
	return fmt.Sprintf("%s:%d col %d", p.Message, p.Line, p.Char)
}

type Footer struct {
	Token string
	Value string
}

type Reference map[string]string

type References []Reference

type Reverts map[string]string

type Notes map[string]string
