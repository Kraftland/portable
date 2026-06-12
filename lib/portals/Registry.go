package portals

import (
	"errors"

	"github.com/godbus/dbus/v5"
)

// Optional further information, none yet
type RegisterOptions struct {

}

// This simple interface lets unsandboxed applications register their D-Bus connections and associate it with an application ID that will be used in portals.
func Register (appID string, options RegisterOptions) (error) {
	bus, err := dbus.SessionBus()
	if err != nil {
		return err
	}

	// TODO: implement proper APP ID checking
	if len(appID) == 0 {
		return errors.New("Invalid application ID: empty")
	}
	portalObj := bus.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)
	optMap := make(map[string]dbus.Variant)
	call := portalObj.Call(
		"org.freedesktop.host.portal.Registry.Register",
		0,
		appID,
		optMap,
	)
	if call.Err != nil {
		return call.Err
	}
	return nil
}