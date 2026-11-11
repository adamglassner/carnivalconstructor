// Carnival Constructor (c) 2026 Ironite.
package main

var version = "0.0.0-dev" // set with -ldflags "-X main.version=..."

const (
	productName = "Carnival Constructor"
	bundleName  = "CarnivalConstructor" // folder under %APPDATA%\Ironite
	gameID      = "carnival"            // Ironite Launcher's id for this game
)

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// money formats dollars with thousands separators: $12,480 or -$300.
func money(n int) string {
	if n < 0 {
		return "-" + money(-n)
	}
	s := itoa(n)
	out := ""
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(r)
	}
	return "$" + out
}
