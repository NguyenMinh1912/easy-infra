package service

import "testing"

func TestParsePortBindings(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []portMapping
	}{
		{"empty", "", nil},
		{"null", "null", nil},
		{
			"single",
			`{"5432/tcp":[{"HostIp":"127.0.0.1","HostPort":"5433"}]}`,
			[]portMapping{{Host: 5433, Container: 5432}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePortBindings(tc.in)
			if err != nil {
				t.Fatalf("parsePortBindings(%q): %v", tc.in, err)
			}
			if !samePorts(got, tc.want) {
				t.Errorf("parsePortBindings(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParsePortBindingsInvalid(t *testing.T) {
	if _, err := parsePortBindings(`{not json}`); err == nil {
		t.Error("parsePortBindings on malformed JSON: want error, got nil")
	}
}

func TestSamePorts(t *testing.T) {
	cases := []struct {
		name string
		a, b []portMapping
		want bool
	}{
		{"both empty", nil, nil, true},
		{
			"equal ignoring order",
			[]portMapping{{Host: 5433, Container: 5432}, {Host: 9000, Container: 9000}},
			[]portMapping{{Host: 9000, Container: 9000}, {Host: 5433, Container: 5432}},
			true,
		},
		{
			// The reported bug: container created on 5432, profile now wants 5433.
			"different host port",
			[]portMapping{{Host: 5432, Container: 5432}},
			[]portMapping{{Host: 5433, Container: 5432}},
			false,
		},
		{
			"different length",
			[]portMapping{{Host: 5432, Container: 5432}},
			nil,
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := samePorts(tc.a, tc.b); got != tc.want {
				t.Errorf("samePorts(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
