package main

import (
	"bufio"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func generatePasswdFile(config Config) {
	user, err := user.Current()
	if err != nil {
		pecho("warn", "Could not get current user info")
		return
	}
	shell := "/usr/bin/bash"
	file, err := os.OpenFile(
		filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "passwd"),
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0700,
	)
	if err != nil {
		pecho("warn", "Could not open fake passwd file")
		return
	}
	defer file.Close()
	builder := strings.Builder{}

	// Own user
	builder.WriteString(user.Username)
	builder.WriteString(":x:")
	builder.WriteString(user.Uid)
	builder.WriteString(":")
	builder.WriteString(user.Gid)
	builder.WriteString(":")
	builder.WriteString(user.Name)
	builder.WriteString(":")
	builder.WriteString(filepath.Join(xdgDir.dataDir, config.Metadata.StateDirectory))
	builder.WriteString(":" + shell)
	builder.WriteString("\n")

	// Overflow user
	builder.WriteString("nobody:x:65534:65534:Kernel Overflow User:/:/usr/bin/nologin")
	builder.WriteString("\n")

	writer := bufio.NewWriter(file)
	_, err = io.Copy(writer, strings.NewReader(builder.String()))
	if err != nil {
		pecho("warn", "Could not write fake passwd file")
		return
	}
	err = writer.Flush()
	if err != nil {
		pecho("warn", "Could not write fake passwd file")
		return
	}
}