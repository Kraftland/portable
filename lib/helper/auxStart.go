package main

import (
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
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
		var wg sync.WaitGroup
		wg.Go(func() {
			if len(filesExpose) % 2 != 0 && len(filesExpose) != 0 {
				warn.Println("Unable to start auxiliary instance: uneven expose map")
			}
			var skip bool
			for idx := range filesExpose {
				if skip {
					continue
				}
				filemapAdd(filesExpose[idx], filesExpose[idx + 1])
				skip = true
			}
		})
		var notif = notification{
			notif:	make(map[string]dbus.Variant),
		}
		notif.notif["title"] = dbus.MakeVariant("Failed to start auxiliary instance - Portable")
		iconMap := make(map[string]dbus.Variant)
		iconMap["themed"] = dbus.MakeVariant([]string{"sad-computer-symbolic"})
		notif.notif["icon"] = dbus.MakeVariant(iconMap)
		var buttMap []map[string]dbus.Variant
		buttMap = append(buttMap, map[string]dbus.Variant{
			"action":	dbus.MakeVariant("terminate"),
			"label":	dbus.MakeVariant("Terminate app"),
		})
		notif.notif["priority"] = dbus.MakeVariant("high")
		notif.notif["buttons"] = dbus.MakeVariant(buttMap)
		notif.notif["display-hint"] = dbus.MakeVariant([]string{"persistent", "show-as-new"})
		path := os.Getenv("XDG_RUNTIME_DIR")
		if len(path) == 0 {
			warn.Println("XDG_RUNTIME_DIR not set")
			return
		}
		var req StartNofifyMsg
		if tray {
			warn.Println("Tray activation not supported yet")
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
				warn.Println("Could not pick temp dir")
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
					warn.Println("Could not create directory for stream: " + err.Error())
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
			warn.Println("Could not resolve address: " + err.Error())
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
			warn.Println("Could not stream command:", err)
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
			warn.Println("Could not resolve address: " + err.Error())
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
			warn.Println("Could not stream command:", err)
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
			warn.Println("Could not resolve address: " + err.Error())
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
			warn.Println("Could not stream command:", err)
			notif.notif["body"] = dbus.MakeVariant("Could not stream command: " + err.Error())
			err := addNotif(notif, []string{"terminate"})
			if err != nil {
				panic(err)
			}
			terminateNotify <- 1
			return
		}
		wg.Wait()
		cmdline = append(cmdline, args...)
		cmdlineFinal := cmdlineReplacer(cmdline)
		debug.Println("Received start request from D-Bus:", cmdline, "translated to", cmdlineFinal)
		cmd := exec.Command(cmdlineFinal[0], cmdlineFinal[1:]...)
		cmd.SysProcAttr = procAttr
		isStream = true
		req.cmd = cmd
		req.sockDir = sockDir
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
					{
						Name:		"RequestFSAccess",
						Args:		[]introspect.Arg{
							{
								Name:		"Directory",
								Type:		"b",
								Direction:	"in",
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
		default:
			warn.Fatalln("Could not own bus name: " + reply.String())
	}
}