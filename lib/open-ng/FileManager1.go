package main

import (
	"os"

	"github.com/godbus/dbus/v5"
)

func openPathFileManager1 (path string) error {
	warn.Println("Using legacy FileManager1 opening calls, most likely there's a logic error")
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	pathSlice := []string{"file://" + path}
	fileManager1Obj := conn.Object("org.freedesktop.FileManager1", "/org/freedesktop/FileManager1")
	call := fileManager1Obj.Call(
		"org.freedesktop.FileManager1.ShowItems",
		0,
		pathSlice,
		os.Getenv("appID"),
	)
	return call.Err
}
