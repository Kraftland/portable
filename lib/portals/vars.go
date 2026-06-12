package portals

import (
	"github.com/godbus/dbus/v5"
)

type AppID string

type portalResponse struct {
	Response	int
	Results		map[string]dbus.Variant
}

type Iface string

// Required Portal versions
const (
	FileChooserVersion	=	4
)