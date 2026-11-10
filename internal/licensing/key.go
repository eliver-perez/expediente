package licensing

import (
	"regexp"
	"strings"
)

const commercialAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

var commercialCode = regexp.MustCompile(`^[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{3}-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}-[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{4}$`)
var commercialCompact = regexp.MustCompile(`^[23456789ABCDEFGHJKMNPQRSTUVWXYZ]{15}$`)

// NormalizeCommercialKey accepts exactly the two representations in the server
// contract. It never substitutes visually similar characters.
func NormalizeCommercialKey(input string) (string, error) {
	value := strings.ToUpper(strings.Trim(input, " \t\r\n"))
	if commercialCode.MatchString(value) {
		return value, nil
	}
	if commercialCompact.MatchString(value) {
		return value[:3] + "-" + value[3:7] + "-" + value[7:11] + "-" + value[11:], nil
	}
	return "", failure("LICENSE_KEY_INVALID", "Introduce un código de 15 caracteres en formato 3-4-4-4, sin I, L, O, 0 ni 1.")
}
