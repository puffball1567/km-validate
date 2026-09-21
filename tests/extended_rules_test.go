package validate

import "testing"

func TestComparisonRuleMatrix(t *testing.T) {
	tests := []struct {
		name, field, code string
		run               func(*Validator) error
	}{
		{"exact empty zero", "value", "", func(v *Validator) error { return v.ExactLength("value", "", 0) }},
		{"exact Unicode", "value", "", func(v *Validator) error { return v.ExactLength("value", "金木犀", 3) }},
		{"exact too short", "value", "exact_length", func(v *Validator) error { return v.ExactLength("value", "金木犀", 4) }},
		{"equal", "confirmation", "", func(v *Validator) error { return v.Equal("confirmation", "secret", "secret") }},
		{"equal empty", "confirmation", "", func(v *Validator) error { return v.Equal("confirmation", "", "") }},
		{"equal case sensitive", "confirmation", "equal", func(v *Validator) error { return v.Equal("confirmation", "Secret", "secret") }},
		{"not equal", "name", "", func(v *Validator) error { return v.NotEqual("name", "allowed", "reserved") }},
		{"not equal rejected", "name", "not_equal", func(v *Validator) error { return v.NotEqual("name", "reserved", "reserved") }},
		{"contains ASCII", "description", "", func(v *Validator) error { return v.Contains("description", "hello world", "world") }},
		{"contains Unicode", "description", "", func(v *Validator) error { return v.Contains("description", "金木犀の香り", "木犀") }},
		{"contains missing", "description", "contains", func(v *Validator) error { return v.Contains("description", "hello", "world") }},
		{"starts with", "path", "", func(v *Validator) error { return v.StartsWith("path", "/api/users", "/api/") }},
		{"starts with missing", "path", "starts_with", func(v *Validator) error { return v.StartsWith("path", "/admin", "/api/") }},
		{"ends with", "file", "", func(v *Validator) error { return v.EndsWith("file", "report.json", ".json") }},
		{"ends with missing", "file", "ends_with", func(v *Validator) error { return v.EndsWith("file", "report.txt", ".json") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewValidator()
			if err := test.run(validator); err != nil {
				t.Fatalf("rule error = %v", err)
			}
			if test.code == "" {
				if !validator.Valid() {
					t.Fatalf("issues = %#v", validator.All())
				}
			} else {
				requireIssue(t, validator, test.field, test.code)
			}
		})
	}
}

func TestComparisonRuleConfigurationMatrix(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Validator) error
	}{
		{"negative exact length", func(v *Validator) error { return v.ExactLength("field", "value", -1) }},
		{"empty substring", func(v *Validator) error { return v.Contains("field", "value", "") }},
		{"empty prefix", func(v *Validator) error { return v.StartsWith("field", "value", "") }},
		{"empty suffix", func(v *Validator) error { return v.EndsWith("field", "value", "") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewValidator()
			if err := validator.Reject("existing", "existing", "existing issue"); err != nil {
				t.Fatal(err)
			}
			if err := test.run(validator); err == nil {
				t.Fatal("configuration error = nil")
			}
			if got := validator.All(); len(got) != 1 || got[0].Field != "existing" {
				t.Fatalf("configuration error mutated state: %#v", got)
			}
		})
	}
}

func TestStructuredFormatRuleMatrix(t *testing.T) {
	tests := []struct {
		name, field, code string
		run               func(*Validator) error
	}{
		{"IPv4", "address", "", func(v *Validator) error { return v.IpAddress("address", "192.0.2.1") }},
		{"IPv6", "address", "", func(v *Validator) error { return v.IpAddress("address", "2001:db8::1") }},
		{"IPv4 mapped IPv6", "address", "", func(v *Validator) error { return v.IpAddress("address", "::ffff:192.0.2.1") }},
		{"IP with port", "address", "ip_address", func(v *Validator) error { return v.IpAddress("address", "192.0.2.1:80") }},
		{"IP whitespace", "address", "ip_address", func(v *Validator) error { return v.IpAddress("address", " 192.0.2.1") }},
		{"empty IP", "address", "ip_address", func(v *Validator) error { return v.IpAddress("address", "") }},
		{"IPv4 CIDR", "network", "", func(v *Validator) error { return v.Cidr("network", "192.0.2.0/24") }},
		{"IPv6 CIDR", "network", "", func(v *Validator) error { return v.Cidr("network", "2001:db8::/32") }},
		{"CIDR host bits", "network", "", func(v *Validator) error { return v.Cidr("network", "192.0.2.123/24") }},
		{"CIDR missing prefix", "network", "cidr", func(v *Validator) error { return v.Cidr("network", "192.0.2.1") }},
		{"CIDR prefix too large", "network", "cidr", func(v *Validator) error { return v.Cidr("network", "192.0.2.0/33") }},
		{"UUID lowercase", "id", "", func(v *Validator) error { return v.Uuid("id", "550e8400-e29b-41d4-a716-446655440000") }},
		{"UUID uppercase", "id", "", func(v *Validator) error { return v.Uuid("id", "550E8400-E29B-41D4-A716-446655440000") }},
		{"UUID nil", "id", "", func(v *Validator) error { return v.Uuid("id", "00000000-0000-0000-0000-000000000000") }},
		{"UUID short", "id", "uuid", func(v *Validator) error { return v.Uuid("id", "550e8400-e29b") }},
		{"UUID bad hyphen", "id", "uuid", func(v *Validator) error { return v.Uuid("id", "550e84000e29b-41d4-a716-446655440000") }},
		{"UUID bad hex", "id", "uuid", func(v *Validator) error { return v.Uuid("id", "550e8400-e29b-41d4-a716-44665544000z") }},
		{"timestamp UTC", "created_at", "", func(v *Validator) error { return v.Timestamp("created_at", "2026-09-21T01:02:03Z") }},
		{"timestamp offset", "created_at", "", func(v *Validator) error { return v.Timestamp("created_at", "2026-09-21T10:02:03+09:00") }},
		{"timestamp fractional", "created_at", "", func(v *Validator) error { return v.Timestamp("created_at", "2026-09-21T01:02:03.123456789Z") }},
		{"timestamp date only", "created_at", "timestamp", func(v *Validator) error { return v.Timestamp("created_at", "2026-09-21") }},
		{"timestamp missing zone", "created_at", "timestamp", func(v *Validator) error { return v.Timestamp("created_at", "2026-09-21T01:02:03") }},
		{"timestamp invalid date", "created_at", "timestamp", func(v *Validator) error { return v.Timestamp("created_at", "2026-02-30T01:02:03Z") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := NewValidator()
			if err := test.run(validator); err != nil {
				t.Fatalf("rule error = %v", err)
			}
			if test.code == "" {
				if !validator.Valid() {
					t.Fatalf("issues = %#v", validator.All())
				}
			} else {
				requireIssue(t, validator, test.field, test.code)
			}
		})
	}
}

func TestExtendedRulePathErrors(t *testing.T) {
	operations := []func(*Validator) error{
		func(v *Validator) error { return v.ExactLength("bad path", "", 0) },
		func(v *Validator) error { return v.Equal("bad path", "", "") },
		func(v *Validator) error { return v.NotEqual("bad path", "", "") },
		func(v *Validator) error { return v.Contains("bad path", "a", "a") },
		func(v *Validator) error { return v.StartsWith("bad path", "a", "a") },
		func(v *Validator) error { return v.EndsWith("bad path", "a", "a") },
		func(v *Validator) error { return v.IpAddress("bad path", "192.0.2.1") },
		func(v *Validator) error { return v.Cidr("bad path", "192.0.2.0/24") },
		func(v *Validator) error { return v.Uuid("bad path", "550e8400-e29b-41d4-a716-446655440000") },
		func(v *Validator) error { return v.Timestamp("bad path", "2026-09-21T01:02:03Z") },
	}
	for index, operation := range operations {
		validator := NewValidator()
		if err := operation(validator); err == nil {
			t.Fatalf("operation %d error = nil", index)
		}
		if !validator.Valid() {
			t.Fatalf("operation %d mutated state: %#v", index, validator.All())
		}
	}
}
