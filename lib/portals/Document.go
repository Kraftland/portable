package portals

import (
	"context"
	"time"
	"strings"
	dbus "github.com/godbus/dbus/v5"
)

type Document struct {}

// Gets the document portal mount point, times out after 50 milliseconds, and returns the path as a string
func (m *Document) GetMountPoint () (string, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return "", err
	}
	docObj := conn.Object(
		"org.freedesktop.portal.Documents",
		"/org/freedesktop/portal/documents",
	)
	ctx := context.TODO()
	ctxNew, cancelFunc := context.WithTimeout(ctx, 50 * time.Millisecond)
	call := docObj.CallWithContext(
		ctxNew,
		"org.freedesktop.portal.Documents.GetMountPoint",
		0,
	)
	cancelFunc()
	var mntRaw []byte
	if call.Err != nil {
		return "", call.Err
	}
	err := call.Store(&mntRaw)
	if err != nil {
		return "", err
	} else {
		mnt := strings.TrimRight(
			string(mntRaw),
			"\x00",
		)
		return mnt, nil
	}
}