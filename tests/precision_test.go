package validate

import (
	"math"
	"strings"
	"testing"
)

func TestFieldPathMatrix(t *testing.T) {
	valid := []string{"A", "_", "field1", "user.name", "request-id", "a_b.C9-d"}
	for _, field := range valid {
		t.Run("valid_"+field, func(t *testing.T) {
			validator := NewValidator()
			if err := validator.Reject(field, "custom_code2", "message"); err != nil {
				t.Fatalf("Reject(%q) error = %v", field, err)
			}
			requireIssue(t, validator, field, "custom_code2")
		})
	}

	invalid := []string{
		"", " ", " field", "field ", ".field", "field.", "a..b", "1field",
		"a.1field", "bad/name", "bad name", "bad\tname", "名前", "field\x00name",
	}
	for index, field := range invalid {
		t.Run("invalid", func(t *testing.T) {
			validator := NewValidator()
			if err := validator.Reject(field, "code", "message"); err == nil {
				t.Fatalf("case %d Reject(%q) error = nil", index, field)
			}
			if !validator.Valid() {
				t.Fatalf("case %d mutated validator: %#v", index, validator.All())
			}
		})
	}
}

func TestIssueCodeMatrix(t *testing.T) {
	valid := []string{"a", "required", "http_url", "code2", "a_2_b"}
	for _, code := range valid {
		validator := NewValidator()
		if err := validator.Reject("field", code, "message"); err != nil {
			t.Fatalf("Reject(code=%q) error = %v", code, err)
		}
		requireIssue(t, validator, "field", code)
	}
	invalid := []string{"", "Required", "1code", "_code", "bad-code", "bad code", "code\n", "コード"}
	for _, code := range invalid {
		validator := NewValidator()
		if err := validator.Reject("field", code, "message"); err == nil {
			t.Fatalf("Reject(code=%q) error = nil", code)
		}
		if validator.Size() != 0 {
			t.Fatalf("Reject(code=%q) mutated validator", code)
		}
	}
}

func TestIssueCollectionQueriesAndAtomicAddAll(t *testing.T) {
	validator := NewValidator()
	batch := []Issue{
		{Field: "user.name", Code: "required", Message: "name is required"},
		{Field: "user.name", Code: "min_length", Message: "name is too short"},
		{Field: "user.email", Code: "email", Message: "email is invalid"},
	}
	if err := validator.AddAll(batch); err != nil {
		t.Fatal(err)
	}
	batch[0] = Issue{Field: "mutated", Code: "changed", Message: "changed"}
	if validator.Size() != 3 {
		t.Fatalf("Size() = %d", validator.Size())
	}
	if err := validator.AddAll(nil); err != nil || validator.Size() != 3 {
		t.Fatalf("AddAll(empty) = error:%v size:%d", err, validator.Size())
	}
	nameIssues, err := validator.ForField("user.name")
	if err != nil || len(nameIssues) != 2 || nameIssues[0].Code != "required" || nameIssues[1].Code != "min_length" {
		t.Fatalf("ForField(user.name) = (%#v, %v)", nameIssues, err)
	}
	nameIssues[0] = Issue{Field: "mutated", Code: "mutated", Message: "mutated"}
	nameIssues, err = validator.ForField("user.name")
	if err != nil || nameIssues[0].Code != "required" {
		t.Fatalf("ForField() did not return an independent slice: %#v", nameIssues)
	}
	missing, err := validator.ForField("missing")
	if err != nil || len(missing) != 0 {
		t.Fatalf("ForField(missing) = (%#v, %v)", missing, err)
	}
	for _, test := range []struct {
		field string
		want  bool
	}{
		{"user.name", true},
		{"missing", false},
	} {
		got, err := validator.Has(test.field)
		if err != nil || got != test.want {
			t.Fatalf("Has(%q) = (%v, %v), want %v", test.field, got, err, test.want)
		}
	}
	if _, err := validator.ForField("bad path"); err == nil {
		t.Fatal("ForField(invalid) error = nil")
	}
	if _, err := validator.Has("bad path"); err == nil {
		t.Fatal("Has(invalid) error = nil")
	}

	before := validator.All()
	invalidBatches := [][]Issue{
		{{Field: "ok", Code: "ok", Message: "ok"}, {Field: "bad path", Code: "bad", Message: "bad"}},
		{{Field: "field", Code: "Bad", Message: "bad"}},
		{{Field: "field", Code: "bad", Message: "  "}},
	}
	for index, invalid := range invalidBatches {
		if err := validator.AddAll(invalid); err == nil {
			t.Fatalf("AddAll case %d error = nil", index)
		}
		if got := validator.All(); len(got) != len(before) {
			t.Fatalf("AddAll case %d partially mutated: %#v", index, got)
		}
	}
}

func TestNumberRangeMatrix(t *testing.T) {
	tests := []struct {
		name                    string
		value, minimum, maximum float64
		code                    string
	}{
		{"lower boundary", -1, -1, 1, ""},
		{"inside", 0.25, -1, 1, ""},
		{"upper boundary", 1, -1, 1, ""},
		{"below", -1.0001, -1, 1, "number_range"},
		{"above", 1.0001, -1, 1, "number_range"},
		{"nan", math.NaN(), -1, 1, "finite_number"},
		{"positive infinity", math.Inf(1), -1, 1, "finite_number"},
		{"negative infinity", math.Inf(-1), -1, 1, "finite_number"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewValidator()
			if err := validator.NumberRange("ratio", test.value, test.minimum, test.maximum); err != nil {
				t.Fatalf("NumberRange() error = %v", err)
			}
			if test.code == "" {
				if !validator.Valid() {
					t.Fatalf("issues = %#v", validator.All())
				}
			} else {
				requireIssue(t, validator, "ratio", test.code)
			}
		})
	}

	configurations := [][2]float64{
		{math.NaN(), 1}, {0, math.NaN()}, {math.Inf(-1), 1}, {0, math.Inf(1)}, {2, 1},
	}
	for index, bounds := range configurations {
		validator := NewValidator()
		if err := validator.NumberRange("ratio", 0, bounds[0], bounds[1]); err == nil {
			t.Fatalf("configuration %d error = nil", index)
		}
		if !validator.Valid() {
			t.Fatalf("configuration %d mutated: %#v", index, validator.All())
		}
	}
	if err := NewValidator().NumberRange("bad path", 0, 0, 1); err == nil {
		t.Fatal("NumberRange(invalid path) error = nil")
	}
}

func TestEmailAndURLAdversarialMatrix(t *testing.T) {
	emails := []struct {
		value string
		valid bool
	}{
		{"user@example.com", true},
		{"first.last+tag@example.co.jp", true},
		{"", false},
		{"user@", false},
		{"@example.com", false},
		{"user example@example.com", false},
		{"User <user@example.com>", false},
		{" user@example.com ", false},
	}
	for _, test := range emails {
		validator := NewValidator()
		if err := validator.Email("email", test.value); err != nil {
			t.Fatal(err)
		}
		if validator.Valid() != test.valid {
			t.Fatalf("Email(%q) valid = %v, want %v", test.value, validator.Valid(), test.valid)
		}
	}

	urls := []struct {
		value string
		valid bool
	}{
		{"http://example.com", true},
		{"https://example.com/path?q=value#fragment", true},
		{"HTTPS://example.com", true},
		{"https://127.0.0.1:8443/path", true},
		{"", false},
		{"/relative", false},
		{"ftp://example.com", false},
		{"https:///missing-host", false},
		{"https://example.com/%zz", false},
	}
	for _, test := range urls {
		validator := NewValidator()
		if err := validator.HttpURL("site", test.value); err != nil {
			t.Fatal(err)
		}
		if validator.Valid() != test.valid {
			t.Fatalf("HttpURL(%q) valid = %v, want %v, issues=%#v", test.value, validator.Valid(), test.valid, validator.All())
		}
	}
}

func TestConfigurationErrorsNeverMutateExistingState(t *testing.T) {
	validator := NewValidator()
	if err := validator.Reject("existing", "existing", "existing issue"); err != nil {
		t.Fatal(err)
	}
	operations := []func() error{
		func() error { return validator.MinLength("field", "value", -1) },
		func() error { return validator.MaxLength("field", "value", -1) },
		func() error { return validator.LengthRange("field", "value", 2, 1) },
		func() error { return validator.Matches("field", "value", "[") },
		func() error { return validator.OneOf("field", "value", nil) },
		func() error { return validator.IntegerRange("field", 1, 2, 1) },
		func() error { return validator.NumberRange("field", 1, 2, 1) },
	}
	for index, operation := range operations {
		if err := operation(); err == nil {
			t.Fatalf("operation %d error = nil", index)
		}
		if got := validator.All(); len(got) != 1 || got[0].Field != "existing" {
			t.Fatalf("operation %d mutated state: %#v", index, got)
		}
	}
}

func TestCustomMessagesArePreserved(t *testing.T) {
	validator := NewValidator()
	if err := validator.Reject("field", "custom", "line one\nline two"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(validator.All()[0].Message, "\n") {
		t.Fatal("custom messages should be preserved verbatim")
	}
}
