package portals

import (
	dbus "github.com/godbus/dbus/v5"
	"context"
	"time"
)

type PowerProfileMonitor struct{}

// Returns true when low power mode is enabled
func (m *PowerProfileMonitor) PowerSaverEnabled() (bool, error) {
		conn, err := dbus.SessionBus()
		if err != nil {
			return false, err
		}
		busObj := conn.Object(
			"org.freedesktop.portal.Desktop",
			"/org/freedesktop/portal/desktop",
		)
		ctx := context.TODO()
		ctxNew, cancelFunc := context.WithTimeout(ctx, 10 * time.Millisecond)

		call := busObj.CallWithContext(
			ctxNew,
			"org.freedesktop.DBus.Properties.Get",
			dbus.FlagNoAutoStart,
			"org.freedesktop.portal.PowerProfileMonitor",
			"power-saver-enabled",
		)
		cancelFunc()
		if call.Err != nil {
			return false, call.Err
		}
		var powerSave bool
		err = call.Store(&powerSave)
		if err != nil {
			return false, err
		}
		return powerSave, nil
}