package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// The desktop setup form passes credentials through a redirected handle.
// The interactive CLI still requires a terminal and keeps its existing reader.
func readBootstrapForm(input io.Reader) (string, string, string, error) {
	const limit = 8192
	data, err := io.ReadAll(io.LimitReader(input, limit+1))
	defer clear(data)
	var form struct {
		Username     string `json:"username"`
		DisplayName  string `json:"display_name"`
		Password     string `json:"password"`
		Confirmation string `json:"confirmation"`
	}
	if err != nil || len(data) > limit || json.Unmarshal(data, &form) != nil {
		return "", "", "", fmt.Errorf("no se pudieron leer los datos del asistente")
	}
	if form.Password != form.Confirmation {
		return "", "", "", fmt.Errorf("las contraseñas no coinciden")
	}
	return strings.TrimSpace(form.Username), strings.TrimSpace(form.DisplayName), form.Password, nil
}
