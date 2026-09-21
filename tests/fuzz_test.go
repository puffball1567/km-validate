package validate

import "testing"

func FuzzRejectIsAtomic(f *testing.F) {
	f.Add("user.name", "custom_code", "message")
	f.Add("bad path", "Bad-Code", "")
	f.Add("", "", "")
	f.Fuzz(func(t *testing.T, field, code, message string) {
		validator := NewValidator()
		err := validator.Reject(field, code, message)
		if err != nil {
			if validator.Size() != 0 {
				t.Fatalf("failed Reject mutated state: %#v", validator.All())
			}
			return
		}
		issues := validator.All()
		if len(issues) != 1 || issues[0].Field != field || issues[0].Code != code || issues[0].Message != message {
			t.Fatalf("successful Reject = %#v", issues)
		}
	})
}

func FuzzTextRulesNeverPanic(f *testing.F) {
	f.Add("field", "金木犀", 2, 4)
	f.Add("field", "", 0, 0)
	f.Add("bad path", "value", -1, 1)
	f.Fuzz(func(t *testing.T, field, value string, minimum, maximum int) {
		validators := []*Validator{NewValidator(), NewValidator(), NewValidator(), NewValidator()}
		_ = validators[0].Required(field, value)
		_ = validators[1].MinLength(field, value, minimum)
		_ = validators[2].MaxLength(field, value, maximum)
		_ = validators[3].LengthRange(field, value, minimum, maximum)
		for index, validator := range validators {
			if validator.Size() > 1 {
				t.Fatalf("rule %d recorded multiple issues: %#v", index, validator.All())
			}
		}
	})
}

func FuzzStructuredRulesNeverPanic(f *testing.F) {
	f.Add("field", "user@example.com", "https://example.com", "value", ".*")
	f.Add("field", "", "", "", "[")
	f.Add("bad path", "x", "y", "z", "")
	f.Fuzz(func(t *testing.T, field, emailValue, urlValue, value, pattern string) {
		emailValidator := NewValidator()
		urlValidator := NewValidator()
		patternValidator := NewValidator()
		_ = emailValidator.Email(field, emailValue)
		_ = urlValidator.HttpURL(field, urlValue)
		_ = patternValidator.Matches(field, value, pattern)
		for index, validator := range []*Validator{emailValidator, urlValidator, patternValidator} {
			if validator.Size() > 1 {
				t.Fatalf("rule %d recorded multiple issues: %#v", index, validator.All())
			}
		}
	})
}

func FuzzAddAllIsAtomic(f *testing.F) {
	f.Add("first", "first_code", "first message", "second", "second_code", "second message")
	f.Add("valid", "valid", "valid", "bad path", "Bad", "")
	f.Fuzz(func(t *testing.T, firstField, firstCode, firstMessage, secondField, secondCode, secondMessage string) {
		validator := NewValidator()
		err := validator.AddAll([]Issue{
			{Field: firstField, Code: firstCode, Message: firstMessage},
			{Field: secondField, Code: secondCode, Message: secondMessage},
		})
		if err != nil {
			if validator.Size() != 0 {
				t.Fatalf("failed AddAll partially mutated: %#v", validator.All())
			}
			return
		}
		if validator.Size() != 2 {
			t.Fatalf("successful AddAll size = %d", validator.Size())
		}
	})
}
