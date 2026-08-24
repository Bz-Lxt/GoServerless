package model

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTPStatusMapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{NotFound("function", "x"), http.StatusNotFound},
		{Invalid("bad"), http.StatusBadRequest},
		{Unauthorized("no"), http.StatusUnauthorized},
		{Timeout("t"), http.StatusGatewayTimeout},
		{Concurrency("f"), http.StatusTooManyRequests},
		{errors.New("x"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		if got := HTTPStatus(c.err); got != c.want {
			t.Fatalf("%v => %d want %d", c.err, got, c.want)
		}
	}
}

func TestUnwrap(t *testing.T) {
	err := NotFound("function", "n")
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("expected ErrNotFound")
	}
}
