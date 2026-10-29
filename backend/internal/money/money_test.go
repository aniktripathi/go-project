package money

import "testing"

func TestParse(t *testing.T) {
	valid := map[string]Money{
		"1":       100,
		"1.5":     150,
		"1.50":    150,
		"0.01":    1,
		".5":      50,
		"1234.56": 123456,
		" 10 ":    1000,
	}
	for in, want := range valid {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	invalid := []string{"", "abc", "-5", "+5", "0", "0.00", "1.234", "1e3", "1.2.3", ".", "1,000", "9999999999999"}
	for _, in := range invalid {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) should fail", in)
		}
	}
}

func TestString(t *testing.T) {
	cases := map[Money]string{123450: "1234.50", 5: "0.05", 0: "0.00", -5000: "-50.00", 100: "1.00"}
	for in, want := range cases {
		if got := in.String(); got != want {
			t.Errorf("Money(%d).String() = %q; want %q", int64(in), got, want)
		}
	}
}

func TestJSONRoundTrip(t *testing.T) {
	b, _ := Money(123450).MarshalJSON()
	if string(b) != "1234.50" {
		t.Fatalf("marshal = %s", b)
	}
	var m Money
	if err := m.UnmarshalJSON([]byte(`"99.99"`)); err != nil || m != 9999 {
		t.Fatalf("unmarshal = %v, %v", m, err)
	}
}
