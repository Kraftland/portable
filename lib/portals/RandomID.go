package portals

import (
	"strconv"
	"math/rand"
)

// Generates a request token, see https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.Request.html
func GenerateRequestToken() string {
	return "portablePortal" + strconv.Itoa(rand.Int())
}