package main

import (
	"os"
	"strings"
	"path/filepath"
	"bufio"
	"io"
)

func generateNsswitch (config Config) {
	builder := strings.Builder{}
	builder.WriteString("passwd: files\n")
	builder.WriteString("group: files\n")
	builder.WriteString("shadow: files\n")
	builder.WriteString("gshadow: files\n")
	builder.WriteString("publickey: files\n")
	builder.WriteString("hosts: files myhostname resolve [!UNAVAIL=return] dns\n")
	builder.WriteString("networks: files\n")
	builder.WriteString("protocols: files\n")
	builder.WriteString("services: files\n")
	builder.WriteString("ethers: files\n")
	builder.WriteString("rpc: files\n")
	builder.WriteString("netgroup: files\n")

	file, err := os.OpenFile(
		filepath.Join(xdgDir.runtimeDir, "portable", config.Metadata.AppID, "nsswitch"),
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0700,
	)
	if err != nil {
		pecho("warn", "Could not open fake nsswitch file")
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = io.Copy(writer, strings.NewReader(builder.String()))
	if err != nil {
		pecho("warn", "Could not write fake nsswitch file")
		return
	}
	err = writer.Flush()
	if err != nil {
		pecho("warn", "Could not write fake nsswitch file")
		return
	}
}