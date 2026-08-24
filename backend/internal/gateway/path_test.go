package gateway

import "testing"

func TestStripInvokePrefix(t *testing.T) {
	cases := map[string]string{
		"/api/v1/run/hello":           "/",
		"/api/v1/run/hello/":          "/",
		"/api/v1/run/hello/users":     "/users",
		"/api/v1/run/hello/users/1":   "/users/1",
		"/other":                      "/other",
	}
	for in, want := range cases {
		got := StripInvokePrefix(in, "hello")
		if got != want {
			t.Fatalf("%s => %s want %s", in, got, want)
		}
	}
}
