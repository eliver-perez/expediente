package main

import (
	"strings"
	"testing"
)

func TestWindowsBootstrapFormInput(t *testing.T) {
	user, name, secret, err := readBootstrapForm(strings.NewReader(`{"username":" admin ","display_name":"María","password":"prueba ñ","confirmation":"prueba ñ"}`))
	if err != nil || user != "admin" || name != "María" || secret != "prueba ñ" {
		t.Fatal("form input did not preserve credentials and Unicode")
	}
	for _, input := range []string{
		`{"password":"password","confirmation":"different"}`,
		`{"password":"password","confirmation":"password"}{}`,
		strings.Repeat(" ", 8193),
	} {
		if _, _, _, err = readBootstrapForm(strings.NewReader(input)); err == nil {
			t.Fatal("accepted invalid form input")
		}
	}
}
