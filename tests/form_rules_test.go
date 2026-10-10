package validate

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestConditionalRequiredRules(t *testing.T) {
	for _, value := range []string{"", " \t\n", "value"} {
		for _, condition := range []bool{false, true} {
			for _, unless := range []bool{false, true} {
				v := NewValidator()
				var err error
				if unless {
					err = v.RequiredUnless("field", value, condition)
				} else {
					err = v.RequiredIf("field", value, condition)
				}
				if err != nil {
					t.Fatal(err)
				}
				want := value != "value" && (condition != unless)
				if (!v.Valid()) != want {
					t.Fatalf("value %q condition %v unless %v: %v", value, condition, unless, v.All())
				}
				if want {
					code := "required_if"
					if unless {
						code = "required_unless"
					}
					requireIssue(t, v, "field", code)
				}
			}
		}
	}
	for _, others := range [][]string{{"", " "}, {"", "present"}, {"present", ""}, {"present", "other"}} {
		for _, value := range []string{"", "value"} {
			for _, without := range []bool{false, true} {
				v := NewValidator()
				var err error
				if without {
					err = v.RequiredWithout("field", value, others)
				} else {
					err = v.RequiredWith("field", value, others)
				}
				if err != nil {
					t.Fatal(err)
				}
				trigger := false
				for _, other := range others {
					blank := other == "" || other == " "
					if blank == without {
						trigger = true
					}
				}
				if (!v.Valid()) != (trigger && value == "") {
					t.Fatalf("value %q others %v without %v: %v", value, others, without, v.All())
				}
				if !v.Valid() {
					code := "required_with"
					if without {
						code = "required_without"
					}
					requireIssue(t, v, "field", code)
				}
			}
		}
	}
}

func TestCrossFieldComparisons(t *testing.T) {
	for _, same := range []bool{false, true} {
		for _, value := range []string{"", "金木犀", "different"} {
			v := NewValidator()
			var err error
			if same {
				err = v.Same("confirmation", value, "password", "金木犀")
			} else {
				err = v.Different("confirmation", value, "password", "金木犀")
			}
			if err != nil {
				t.Fatal(err)
			}
			if v.Valid() != ((value == "金木犀") == same) {
				t.Fatalf("same %v value %q issues %v", same, value, v.All())
			}
			if !v.Valid() {
				code := "different"
				if same {
					code = "same"
				}
				requireIssue(t, v, "confirmation", code)
			}
		}
	}
	for _, operator := range []string{"<", "<=", ">", ">=", "==", "!="} {
		for _, value := range []int64{math.MinInt64, 0, math.MaxInt64} {
			want := false
			switch operator {
			case "<":
				want = value < 0
			case "<=":
				want = value <= 0
			case ">":
				want = value > 0
			case ">=":
				want = value >= 0
			case "==":
				want = value == 0
			case "!=":
				want = value != 0
			}
			v := NewValidator()
			if err := v.CompareInteger("end", value, operator, "start", 0); err != nil {
				t.Fatal(err)
			}
			if v.Valid() != want {
				t.Fatalf("integer %d %s 0: %v", value, operator, v.All())
			}
			if !want {
				requireIssue(t, v, "end", "compare_integer")
			}
			v = NewValidator()
			if err := v.CompareNumber("end", float64(value), operator, "start", 0); err != nil {
				t.Fatal(err)
			}
			if v.Valid() != want {
				t.Fatalf("number %d %s 0: %v", value, operator, v.All())
			}
			if !want {
				requireIssue(t, v, "end", "compare_number")
			}
		}
	}
	// Integers above JavaScript's precision boundary must not be rounded to float64.
	v := NewValidator()
	if err := v.CompareInteger("end", 9007199254740993, ">", "start", 9007199254740992); err != nil || !v.Valid() {
		t.Fatalf("precision: %v %v", err, v.All())
	}
	for _, value := range []float64{math.NaN(), math.Inf(-1), math.Inf(1)} {
		v := NewValidator()
		if err := v.CompareNumber("end", value, ">", "start", 0); err != nil {
			t.Fatal(err)
		}
		requireIssue(t, v, "end", "finite_number")
	}
}

func TestArrayRulesAndAtomicRollback(t *testing.T) {
	for _, test := range []struct {
		count, min, max int
		valid           bool
	}{{0, 0, 0, true}, {1, 1, 3, true}, {3, 1, 3, true}, {0, 1, 3, false}, {4, 1, 3, false}} {
		v := NewValidator()
		if err := v.ArrayLength("items", test.count, test.min, test.max); err != nil {
			t.Fatal(err)
		}
		if v.Valid() != test.valid {
			t.Fatalf("%+v: %v", test, v.All())
		}
		if !test.valid {
			requireIssue(t, v, "items", "array_length")
		}
	}
	v := NewValidator()
	rule := func(v *Validator, field, value string) error { return v.Email(field, value) }
	if err := each(v, "users", []string{"a@example.com", "bad", "also bad"}, rule); err != nil {
		t.Fatal(err)
	}
	if got := v.All(); len(got) != 2 || got[0].Field != "users.1" || got[1].Field != "users.2" {
		t.Fatalf("indexed issues: %v", got)
	}
	for _, field := range []string{"users.1", "users.2"} {
		has, err := v.Has(field)
		issues, err2 := v.ForField(field)
		if err != nil || err2 != nil || !has || len(issues) != 1 {
			t.Fatalf("query %s: %v %v %v %v", field, has, issues, err, err2)
		}
	}
	before := v.All()
	sentinel := errors.New("backend unavailable")
	err := each(v, "items", []string{"", "fail"}, func(v *Validator, field, value string) error {
		if value == "fail" {
			return sentinel
		}
		return v.Required(field, value)
	})
	if !errors.Is(err, sentinel) || !reflect.DeepEqual(before, v.All()) {
		t.Fatalf("rollback: %v %v", err, v.All())
	}
	if err := each(v, "items", []string{}, rule); err != nil || !reflect.DeepEqual(before, v.All()) {
		t.Fatalf("empty array: %v", err)
	}
	type row struct{ Emails []string }
	v = NewValidator()
	if err := each(v, "users", []row{{Emails: []string{"bad"}}, {Emails: []string{"ok@example.com", "bad"}}}, func(v *Validator, field string, value row) error { return each(v, field+".emails", value.Emails, rule) }); err != nil {
		t.Fatal(err)
	}
	got := v.All()
	if len(got) != 2 || got[0].Field != "users.0.emails.0" || got[1].Field != "users.1.emails.1" {
		t.Fatalf("nested issues: %v", got)
	}
	v = NewValidator()
	if err := each(v, "counts", []int64{1, 0}, func(v *Validator, field string, value int64) error { return v.IntegerRange(field, value, 1, 5) }); err != nil {
		t.Fatal(err)
	}
	requireIssue(t, v, "counts.1", "integer_range")
}

func TestFileRules(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 13, 10, 26, 10}
	tests := []struct {
		name    string
		content []byte
		max     int64
		types   []string
		code    string
	}{
		{"PNG", png, 8, []string{"text/plain", "IMAGE/PNG"}, ""},
		{"text", []byte("hello"), 5, []string{"text/plain"}, ""},
		{"absent", nil, 8, []string{"image/png"}, "required_file"},
		{"present empty", []byte{}, 0, []string{"text/plain"}, ""},
		{"oversize", png, 7, []string{"image/png"}, "file_size"},
		{"zero limit", png, 0, []string{"image/png"}, "file_size"},
		{"fake PNG", []byte("hello"), 100, []string{"image/png"}, "file_type"},
		{"binary", []byte{0, 1, 2, 3}, 4, []string{"application/octet-stream"}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := NewValidator()
			if err := v.File("avatar", test.content, test.max, test.types); err != nil {
				t.Fatal(err)
			}
			if test.code == "" {
				if !v.Valid() {
					t.Fatal(v.All())
				}
			} else {
				requireIssue(t, v, "avatar", test.code)
			}
		})
	}
}

func TestDatabaseLookups(t *testing.T) {
	for _, found := range []bool{false, true} {
		for _, unique := range []bool{false, true} {
			calls := 0
			lookup := func() (bool, error) {
				calls++
				return found, nil
			}
			v := NewValidator()
			var err error
			if unique {
				err = v.Unique("email", lookup)
			} else {
				err = v.Exists("email", lookup)
			}
			if err != nil || calls != 1 {
				t.Fatalf("lookup calls %d error %v", calls, err)
			}
			if v.Valid() != (found != unique) {
				t.Fatalf("found %v unique %v issues %v", found, unique, v.All())
			}
			if !v.Valid() {
				code := "exists"
				if unique {
					code = "unique"
				}
				requireIssue(t, v, "email", code)
			}
		}
	}
	for _, unique := range []bool{false, true} {
		sentinel := errors.New("database unavailable")
		v := NewValidator()
		_ = v.Required("existing", "")
		before := v.All()
		lookup := func() (bool, error) { return true, sentinel }
		var err error
		if unique {
			err = v.Unique("email", lookup)
		} else {
			err = v.Exists("email", lookup)
		}
		if !errors.Is(err, sentinel) || !reflect.DeepEqual(before, v.All()) {
			t.Fatalf("lookup error: %v issues %v", err, v.All())
		}
	}
	// Update exclusion and tenant scoping are owned by the lookup adapter.
	rows := map[string]string{"current": "a@example.com", "other": "b@example.com"}
	lookup := func(value string) func() (bool, error) {
		return func() (bool, error) {
			for id, email := range rows {
				if id != "current" && email == value {
					return true, nil
				}
			}
			return false, nil
		}
	}
	v := NewValidator()
	if err := v.Unique("email", lookup("a@example.com")); err != nil || !v.Valid() {
		t.Fatalf("self exclusion: %v", err)
	}
	if err := v.Unique("email", lookup("b@example.com")); err != nil {
		t.Fatal(err)
	}
	requireIssue(t, v, "email", "unique")
}

func TestFormRuleConfigurationAtomicity(t *testing.T) {
	operations := []func(*Validator) error{
		func(v *Validator) error { return v.RequiredIf("bad path", "", false) },
		func(v *Validator) error { return v.RequiredUnless("bad path", "", true) },
		func(v *Validator) error { return v.RequiredWith("bad path", "", []string{"x"}) },
		func(v *Validator) error { return v.RequiredWithout("bad path", "", []string{"x"}) },
		func(v *Validator) error { return v.RequiredWith("field", "", nil) },
		func(v *Validator) error { return v.RequiredWithout("field", "", nil) },
		func(v *Validator) error { return v.Same("bad path", "", "other", "") },
		func(v *Validator) error { return v.Same("field", "", "bad path", "") },
		func(v *Validator) error { return v.Different("bad path", "", "other", "") },
		func(v *Validator) error { return v.Different("field", "", "bad path", "") },
		func(v *Validator) error { return v.CompareInteger("bad path", 0, "<", "other", 1) },
		func(v *Validator) error { return v.CompareInteger("field", 0, "<", "bad path", 1) },
		func(v *Validator) error { return v.CompareInteger("field", 0, "???", "other", 1) },
		func(v *Validator) error { return v.CompareNumber("bad path", 0, "<", "other", 1) },
		func(v *Validator) error { return v.CompareNumber("field", 0, "<", "bad path", 1) },
		func(v *Validator) error { return v.CompareNumber("field", 0, "???", "other", 1) },
		func(v *Validator) error { return v.CompareNumber("field", 0, "<", "other", math.NaN()) },
		func(v *Validator) error { return v.CompareNumber("field", 0, "<", "other", math.Inf(1)) },
		func(v *Validator) error { return v.ArrayLength("bad path", 0, 0, 0) },
		func(v *Validator) error { return v.ArrayLength("field", -1, 0, 0) },
		func(v *Validator) error { return v.ArrayLength("field", 0, -1, 0) },
		func(v *Validator) error { return v.ArrayLength("field", 0, 2, 1) },
		func(v *Validator) error {
			return v.Unique("bad path", func() (bool, error) { t.Fatal("lookup must not run"); return false, nil })
		},
		func(v *Validator) error {
			return v.Exists("bad path", func() (bool, error) { t.Fatal("lookup must not run"); return false, nil })
		},
		func(v *Validator) error { return v.Unique("field", nil) },
		func(v *Validator) error { return v.Exists("field", nil) },
		func(v *Validator) error { return v.File("bad path", nil, 0, []string{"text/plain"}) },
		func(v *Validator) error { return v.File("field", nil, -1, []string{"text/plain"}) },
		func(v *Validator) error { return v.File("field", nil, 0, nil) },
		func(v *Validator) error { return v.File("field", nil, 0, []string{"text/plain", "bad"}) },
		func(v *Validator) error { return v.File("field", nil, 0, []string{"text/plain; charset=utf-8"}) },
		func(v *Validator) error { return v.File("field", nil, 0, []string{"image/*"}) },
		func(v *Validator) error { return v.File("field", nil, 0, []string{"["}) },
		func(v *Validator) error { return each(v, "bad path", []string{}, nil) },
		func(v *Validator) error { return each(v, "field", []string{}, nil) },
		func(v *Validator) error {
			return each[string](nil, "field", nil, func(*Validator, string, string) error { return nil })
		},
	}
	for i, operation := range operations {
		v := NewValidator()
		_ = v.Required("existing", "")
		before := v.All()
		if err := operation(v); err == nil {
			t.Fatalf("operation %d did not reject configuration", i)
		}
		if !reflect.DeepEqual(before, v.All()) {
			t.Fatalf("operation %d mutated issues: %v", i, v.All())
		}
	}
}

func TestIndexedPaths(t *testing.T) {
	for _, field := range []string{"items.0", "items.12.email", "matrix.0.1", "items.999999999999999999999999999999999999"} {
		v := NewValidator()
		if err := v.Required(field, ""); err != nil {
			t.Fatalf("%s: %v", field, err)
		}
		requireIssue(t, v, field, "required")
	}
	for _, field := range []string{"items.01", "items.00", "items.1x", "0", "0.items", "items.-1", "items.*", "items.0."} {
		v := NewValidator()
		if err := v.Required(field, ""); err == nil || !v.Valid() {
			t.Fatalf("invalid %s: %v %v", field, err, v.All())
		}
	}
}
