package main

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	// e.g. /org/freedesktop/portal/desktop/request/1_155/gtk769392454
	inhibitRequest string = "phelper"
)

var (
	packageHasInhibit bool
)

func callInhibit(conn *dbus.Conn) {
	if ! packageHasInhibit {
		return
	}
	busObj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	m := make(map[string]dbus.Variant)
	m["handle_token"] = dbus.MakeVariant(inhibitRequest)
	m["reason"] = dbus.MakeVariant("Package requested inhibition")
	var options []map[string]dbus.Variant
	options = append(options, m)
	call := busObj.Call(
		"org.freedesktop.portal.Inhibit.Inhibit",
		0,
		"",
		uint32(12),
		m,
	)
	if call.Err != nil {
		fmt.Println("Could not inhibit suspend:", call.Err)
	}

}