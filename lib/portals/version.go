package portals

import (
	"github.com/godbus/dbus/v5"
)

// Reads the version property of said Portal interface.
func ReadPortalVersion(iface Iface) (int, error) {
	bus, err := dbus.SessionBus()
	if err != nil {
		return 0, err
	}
	object := bus.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)

	call := object.Call(
		"org.freedesktop.DBus.Properties.Get",
		0,
		iface,
		"version",
	)
	if call.Err != nil {
		return 0, call.Err
	}
	var version int
	err = call.Store(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}