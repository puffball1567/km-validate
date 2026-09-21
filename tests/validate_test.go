package validate

import (
	"strings"
	"testing"
)

func requireIssue(t *testing.T, validator *Validator, field, code string) {
	t.Helper()
	issues := validator.All()
	if len(issues) == 0 {
		t.Fatalf("no issue recorded, want %s/%s", field, code)
	}
	issue := issues[len(issues)-1]
	if issue.Field != field || issue.Code != code || strings.TrimSpace(issue.Message) == "" {
		t.Fatalf("issue = %#v, want field=%q code=%q", issue, field, code)
	}
}

func TestValidatorLifecycleAndCustomIssues(t *testing.T) {
	validator := newValidator()
	if !validator.Valid() || validator.Size() != 0 || len(validator.All()) != 0 {
		t.Fatalf("new validator = valid:%v size:%d issues:%#v", validator.Valid(), validator.Size(), validator.All())
	}
	if err := validator.Reject("user.name", "reserved", "name is reserved"); err != nil {
		t.Fatal(err)
	}
	if validator.Valid() || validator.Size() != 1 {
		t.Fatalf("rejected validator = valid:%v size:%d", validator.Valid(), validator.Size())
	}
	issues := validator.All()
	issues[0] = Issue{Field: "mutated"}
	requireIssue(t, validator, "user.name", "reserved")
	validator.Clear()
	if !validator.Valid() || validator.Size() != 0 {
		t.Fatalf("cleared validator = valid:%v size:%d", validator.Valid(), validator.Size())
	}

	for _, test := range []struct {
		name, field, code, message string
	}{
		{"blank field", " ", "code", "message"},
		{"control field", "user\nname", "code", "message"},
		{"blank code", "field", " ", "message"},
		{"blank message", "field", "code", "\t"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validator.Reject(test.field, test.code, test.message); err == nil {
				t.Fatal("Reject() error = nil")
			}
		})
	}
}

func TestTextRuleMatrix(t *testing.T) {
	tests := []struct {
		name, field, code string
		run               func(*Validator) error
	}{
		{"required pass", "name", "", func(v *Validator) error { return v.Required("name", "value") }},
		{"required blank", "name", "required", func(v *Validator) error { return v.Required("name", " \t") }},
		{"min zero empty", "name", "", func(v *Validator) error { return v.MinLength("name", "", 0) }},
		{"min empty fail", "name", "min_length", func(v *Validator) error { return v.MinLength("name", "", 1) }},
		{"min boundary", "name", "", func(v *Validator) error { return v.MinLength("name", "金木犀", 3) }},
		{"min fail", "name", "min_length", func(v *Validator) error { return v.MinLength("name", "金木犀", 4) }},
		{"max zero empty", "name", "", func(v *Validator) error { return v.MaxLength("name", "", 0) }},
		{"max zero fail", "name", "max_length", func(v *Validator) error { return v.MaxLength("name", "a", 0) }},
		{"max boundary", "name", "", func(v *Validator) error { return v.MaxLength("name", "金木犀", 3) }},
		{"max fail", "name", "max_length", func(v *Validator) error { return v.MaxLength("name", "金木犀", 2) }},
		{"range zero empty", "name", "", func(v *Validator) error { return v.LengthRange("name", "", 0, 0) }},
		{"range lower", "name", "", func(v *Validator) error { return v.LengthRange("name", "ab", 2, 4) }},
		{"range upper", "name", "", func(v *Validator) error { return v.LengthRange("name", "abcd", 2, 4) }},
		{"range below", "name", "length_range", func(v *Validator) error { return v.LengthRange("name", "a", 2, 4) }},
		{"range above", "name", "length_range", func(v *Validator) error { return v.LengthRange("name", "abcde", 2, 4) }},
		{"pattern match", "slug", "", func(v *Validator) error { return v.Matches("slug", "hello-42", `^[a-z]+-[0-9]+$`) }},
		{"pattern empty", "slug", "", func(v *Validator) error { return v.Matches("slug", "", `^$`) }},
		{"pattern fail", "slug", "pattern", func(v *Validator) error { return v.Matches("slug", "NO", `^[a-z]+$`) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewValidator()
			if err := test.run(validator); err != nil {
				t.Fatalf("rule error = %v", err)
			}
			if test.code == "" && !validator.Valid() {
				t.Fatalf("issues = %#v", validator.All())
			}
			if test.code != "" {
				requireIssue(t, validator, test.field, test.code)
			}
		})
	}
}

func TestTypedRuleMatrix(t *testing.T) {
	tests := []struct {
		name, field, code string
		run               func(*Validator) error
	}{
		{"email", "email", "", func(v *Validator) error { return v.Email("email", "user@example.com") }},
		{"email malformed", "email", "email", func(v *Validator) error { return v.Email("email", "not an email") }},
		{"email display name", "email", "email", func(v *Validator) error { return v.Email("email", "User <user@example.com>") }},
		{"http", "site", "", func(v *Validator) error { return v.HttpURL("site", "http://example.com/a") }},
		{"https", "site", "", func(v *Validator) error { return v.HttpURL("site", "https://example.com") }},
		{"bad URL", "site", "http_url", func(v *Validator) error { return v.HttpURL("site", "%") }},
		{"bad scheme", "site", "http_url", func(v *Validator) error { return v.HttpURL("site", "ftp://example.com") }},
		{"missing host", "site", "http_url", func(v *Validator) error { return v.HttpURL("site", "/relative") }},
		{"choice first", "mode", "", func(v *Validator) error { return v.OneOf("mode", "dev", []string{"dev", "prod"}) }},
		{"choice later", "mode", "", func(v *Validator) error { return v.OneOf("mode", "prod", []string{"dev", "prod"}) }},
		{"choice miss", "mode", "one_of", func(v *Validator) error { return v.OneOf("mode", "test", []string{"dev", "prod"}) }},
		{"choice empty", "mode", "", func(v *Validator) error { return v.OneOf("mode", "", []string{""}) }},
		{"integer lower", "port", "", func(v *Validator) error { return v.IntegerRange("port", 1, 1, 65535) }},
		{"integer upper", "port", "", func(v *Validator) error { return v.IntegerRange("port", 65535, 1, 65535) }},
		{"integer below", "port", "integer_range", func(v *Validator) error { return v.IntegerRange("port", 0, 1, 65535) }},
		{"integer above", "port", "integer_range", func(v *Validator) error { return v.IntegerRange("port", 65536, 1, 65535) }},
		{"integer full minimum", "number", "", func(v *Validator) error { return v.IntegerRange("number", -1<<63, -1<<63, 1<<63-1) }},
		{"integer full maximum", "number", "", func(v *Validator) error { return v.IntegerRange("number", 1<<63-1, -1<<63, 1<<63-1) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewValidator()
			if err := test.run(validator); err != nil {
				t.Fatalf("rule error = %v", err)
			}
			if test.code == "" && !validator.Valid() {
				t.Fatalf("issues = %#v", validator.All())
			}
			if test.code != "" {
				requireIssue(t, validator, test.field, test.code)
			}
		})
	}
}

func TestRuleConfigurationAndPathErrors(t *testing.T) {
	configuration := []func(*Validator) error{
		func(v *Validator) error { return v.MinLength("field", "value", -1) },
		func(v *Validator) error { return v.MaxLength("field", "value", -1) },
		func(v *Validator) error { return v.LengthRange("field", "value", -1, 1) },
		func(v *Validator) error { return v.LengthRange("field", "value", 0, -1) },
		func(v *Validator) error { return v.LengthRange("field", "value", 2, 1) },
		func(v *Validator) error { return v.Matches("field", "value", "[") },
		func(v *Validator) error { return v.OneOf("field", "value", nil) },
		func(v *Validator) error { return v.IntegerRange("field", 1, 2, 1) },
	}
	for index, run := range configuration {
		validator := NewValidator()
		if err := run(validator); err == nil || !validator.Valid() {
			t.Fatalf("configuration case %d = error:%v issues:%#v", index, err, validator.All())
		}
	}

	paths := []func(*Validator) error{
		func(v *Validator) error { return v.Required("", "value") },
		func(v *Validator) error { return v.MinLength("", "value", 1) },
		func(v *Validator) error { return v.MaxLength("", "value", 1) },
		func(v *Validator) error { return v.LengthRange("", "value", 1, 2) },
		func(v *Validator) error { return v.Matches("", "value", ".*") },
		func(v *Validator) error { return v.Email("", "user@example.com") },
		func(v *Validator) error { return v.HttpURL("", "https://example.com") },
		func(v *Validator) error { return v.OneOf("", "a", []string{"a"}) },
		func(v *Validator) error { return v.IntegerRange("", 1, 0, 2) },
	}
	for index, run := range paths {
		if err := run(NewValidator()); err == nil {
			t.Fatalf("path case %d error = nil", index)
		}
	}
}
