package main

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"fmt"
	"os"
	"math/rand"
	"strconv"
	"path/filepath"
	"net"
	"os/exec"
	"strings"
)

type AuxStartMsg struct {
	CustomTarget	bool
	TargetExec	[]string
	Args		[]string
	ID		int
}

type busStartProcessor struct{
	cmdPfx		[]string
}

func (m *busStartProcessor) AuxStart (
	customTgt bool, tray bool, customExec []string, args []string, filesExpose []string,
	) (
	isStream bool,
	baseDir	string,
	busErr *dbus.Error,
	) {
		var notif = notification{
			notif:	make(map[string]dbus.Variant),
		}
		var butt button
		butt.action = "terminate"
		butt.label = "Terminate app"
		notif.notif["title"] = dbus.MakeVariant("Failed to start auxiliary instance - Portable")
		notif.notif["icon"] = dbus.MakeVariant(icon{
			themed:		[]string{"sad-computer-symbolic"},
		})
		notif.notif["priority"] = dbus.MakeVariant("high")
		notif.notif["buttons"] = dbus.MakeVariant([]button{butt})
		notif.notif["display-hint"] = dbus.MakeVariant([]string{"persistent", "show-as-new"})
		path := os.Getenv("XDG_RUNTIME_DIR")
		if len(path) == 0 {
			fmt.Println("XDG_RUNTIME_DIR not set")
			return
		}
		var req StartNofifyMsg
		if tray {
			fmt.Println("Tray activation not supported yet")
			return
		}
		var cmdline []string
		if customTgt {
			cmdline = customExec
		} else {
			cmdline = m.cmdPfx
		}

		var sockDir string
		var trials int

		for {
			if trials > 512 {
				fmt.Println("Could not pick temp dir")
				notif.notif["body"] = dbus.MakeVariant("Could not pick temp dir")
				err := addNotif(notif, []string{"terminate"})
				if err != nil {
					panic(err)
				}
				terminateNotify <- 1
				return
			}
			trials++
			id := rand.Int()
			idCand := strconv.Itoa(id)
			sockDir = filepath.Join(path, "portable", os.Getenv("appID"), "stream", idCand)

			_, err := os.Stat(sockDir)
			if err != nil {
				err := os.MkdirAll(sockDir, 0700)
				if err != nil {
					fmt.Println("Could not create directory for stream: " + err.Error())
					notif.notif["body"] = dbus.MakeVariant("Could not create directory for stream: " + err.Error())
					err := addNotif(notif, []string{"terminate"})
					if err != nil {
						panic(err)
					}
					terminateNotify <- 1
				} else {
					break
				}
			} else {
				continue
			}
		}

		baseDir = sockDir

		inAddr, err := net.ResolveUnixAddr("unix", filepath.Join(sockDir, "stdin"))
		if err != nil {
			fmt.Println("Could not resolve address: " + err.Error())
			notif.notif["body"] = dbus.MakeVariant("Could not resolve address: " + err.Error())
			err := addNotif(notif, []string{"terminate"})
			if err != nil {
				panic(err)
			}
			terminateNotify <- 1
			return
		}
		stdinListen, err := net.ListenUnix("unix", inAddr)
		if err != nil {
			fmt.Println("Could not stream command:", err)
			notif.notif["body"] = dbus.MakeVariant("Could not stream command: " + err.Error())
			err := addNotif(notif, []string{"terminate"})
			if err != nil {
				panic(err)
			}
			terminateNotify <- 1
			return
		}
		outAddr, err := net.ResolveUnixAddr("unix", filepath.Join(sockDir, "stdout"))
		if err != nil {
			fmt.Println("Could not resolve address: " + err.Error())
			notif.notif["body"] = dbus.MakeVariant("Could not resolve address: " + err.Error())
			err := addNotif(notif, []string{"terminate"})
			if err != nil {
				panic(err)
			}
			terminateNotify <- 1
			return
		}
		stdoutListen, err := net.ListenUnix("unix", outAddr)
		if err != nil {
			fmt.Println("Could not stream command:", err)
			notif.notif["body"] = dbus.MakeVariant("Could not stream command: " + err.Error())
			err := addNotif(notif, []string{"terminate"})
			if err != nil {
				panic(err)
			}
			terminateNotify <- 1
			return
		}
		errAddr, err := net.ResolveUnixAddr("unix", filepath.Join(sockDir, "stderr"))
		if err != nil {
			fmt.Println("Could not resolve address: " + err.Error())
			notif.notif["body"] = dbus.MakeVariant("Could not resolve address: " + err.Error())
			err := addNotif(notif, []string{"terminate"})
			if err != nil {
				panic(err)
			}
			terminateNotify <- 1
			return
		}
		stderrListen, err := net.ListenUnix("unix", errAddr)
		if err != nil {
			fmt.Println("Could not stream command:", err)
			notif.notif["body"] = dbus.MakeVariant("Could not stream command: " + err.Error())
			err := addNotif(notif, []string{"terminate"})
			if err != nil {
				panic(err)
			}
			terminateNotify <- 1
			return
		}

		cmdline = append(cmdline, args...)
		replacer := strings.NewReplacer(filesExpose...)
		cmdlineFinal := []string{}
		for _, cmd := range cmdline {
			cmdlineFinal = append(cmdlineFinal, replacer.Replace(cmd))
		}
		fmt.Println("Received start request from D-Bus:", cmdline)
		cmd := exec.Command(cmdlineFinal[0], cmdlineFinal[1:]...)
		cmd.SysProcAttr = procAttr
		isStream = true
		req.cmd = cmd
		req.UDS = []net.Listener{stdinListen, stdoutListen, stderrListen}
		startNotifier <- req
		return
	}

func busAuxStart(conn *dbus.Conn, cmdPfx []string) {
	proc := new(busStartProcessor)
	proc.cmdPfx = cmdPfx
	var objPath = "/top/kimiblock/portable/init"
	var busName = os.Getenv("appID") + ".Portable.Helper"

	err := conn.Export(proc, dbus.ObjectPath(objPath), "top.kimiblock.Portable.Init")
	if err != nil {
		panic(err)
	}

	node := &introspect.Node{
		Interfaces:	[]introspect.Interface{
			{
				Name:		"top.kimiblock.Portable.Init",
				Methods:	[]introspect.Method{
					{
						Name:		"AuxStart",
						Args:		[]introspect.Arg{
							{
								Name:		"CustomTarget",
								Type:		"b",
								Direction:	"in",
							},
							{
								Name:		"TrayActivate",
								Type:		"b",
								Direction:	"in",
							},
							{
								Name:		"TargetExec",
								Type:		"as",
								Direction:	"in",
							},
							{
								Name:		"Args",
								Type:		"as",
								Direction:	"in",
							},
							{
								Name:		"ExtraFiles",
								Type:		"as",
								Direction:	"in",
							},
							{
								Name:		"IsStream",
								Type:		"b",
								Direction:	"out",
							},
							{
								Name:		"BaseDir",
								Type:		"s",
								Direction:	"out",
							},
						},
					},
				},
			},
		},
	}
	conn.Export(introspect.NewIntrospectable(node), dbus.ObjectPath(objPath), "org.freedesktop.DBus.Introspectable")
	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		panic(err)
	}
	switch reply {
		case dbus.RequestNameReplyPrimaryOwner:
			fmt.Println("Successfully owned bus name")
		default:
			fmt.Println("Could not own bus name: " + reply.String())
			os.Exit(1)
	}
}