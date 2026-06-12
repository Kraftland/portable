package portals

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"net/url"

	"github.com/godbus/dbus/v5"
)

// The options struct for FileChooser()
type FileChooserOptions struct {
	ParentWindow		string
	Title			string

	AcceptLabel		string
	Multiple		bool
	Directory		bool
	CurrentFolder		string
}

// Requests the user to open a file. Directory is not supported. You may need to register with Register() if not sandboxed. This function blocks until UI ends.
// Output will contain the URIs for files with file:// stripped
func FileChooser (options FileChooserOptions) ([]string, error) {
	ver, err := ReadPortalVersion("org.freedesktop.portal.FileChooser")
	if err != nil {
		return []string{}, err
	}
	if ver < FileChooserVersion {
		return []string{}, errors.New(
			"Unsupported Portal version " + strconv.Itoa(ver) + ", minimum: " + strconv.Itoa(FileChooserVersion),
		)
	}

	var wg sync.WaitGroup
	id := GenerateRequestToken()
	conn, err := dbus.SessionBus()
	if err != nil {
		return []string{}, err
	}

	busname := conn.Names()[0]
	absName := strings.ReplaceAll(strings.TrimPrefix(busname, ":"), ".", "_")
	var objPath string = filepath.Join("/org/freedesktop/portal/desktop/request", absName, id)
	err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
		dbus.WithMatchObjectPath(dbus.ObjectPath(objPath)),
		dbus.WithMatchSender("org.freedesktop.portal.Desktop"),
	)
	if err != nil {
		return []string{}, err
	}

	var resultChan = make(chan portalResponse, 1)

	wg.Add(1)
	go func () {
		sigChan := make(chan *dbus.Signal, 512)
		conn.Signal(sigChan)
		wg.Done()
		for sig := range sigChan {
			if sig.Path == dbus.ObjectPath(objPath) && sig.Name == "org.freedesktop.portal.Request.Response" {
			} else {
				continue
			}
			var response portalResponse
			err := dbus.Store(sig.Body, &response.Response, &response.Results)
			if err != nil {
				panic(err)
			}
			resultChan <- response
		}
	} ()
	portalObj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	portalOptions := make(map[string]dbus.Variant)
	portalOptions["handle_token"] = dbus.MakeVariant(id)
	portalOptions["multiple"] = dbus.MakeVariant(options.Multiple)
	portalOptions["directory"] = dbus.MakeVariant(options.Directory)
	if len(options.CurrentFolder) > 0 {
		var bytes []byte
		bytes = []byte(options.CurrentFolder)
		portalOptions["current_folder"] = dbus.MakeVariant(bytes)
	}
	if len(options.AcceptLabel) > 0 {
		portalOptions["accept_label"] = dbus.MakeVariant(options.AcceptLabel)
	}

	wg.Wait()
	call := portalObj.Call(
		"org.freedesktop.portal.FileChooser.OpenFile",
		dbus.FlagAllowInteractiveAuthorization,
		options.ParentWindow,
		options.Title,
		portalOptions,
	)
	if call.Err != nil {
		return []string{}, call.Err
	}
	response := <- resultChan
	switch response.Response {
		case 0:
			// Access granted by user
		case 1:
			return []string{}, errors.New("User interaction cancelled")
		case 2:
			return []string{}, errors.New("The user interaction was ended in some other way")
		default:
			return []string{}, errors.New("Unknown response status " + strconv.Itoa(int(response.Response)))
	}

	// Get URIs
	var uris []string
	val, ok := response.Results["uris"]
	if ok {
		err := val.Store(&uris)
		if err != nil {
			return []string{}, err
		}
		if len(uris) == 0 {
			return []string{}, nil
		}
	} else {
		return []string{}, errors.New("Did not receive any URI to share")
	}
	for idx, val := range uris {
		valTmp, _ := strings.CutPrefix(val, "file://")
		uris[idx], _ = url.PathUnescape(valTmp)
	}

	return uris, nil
}