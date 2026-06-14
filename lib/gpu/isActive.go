package gpu

import (
	"sync"

	udev "github.com/jochenvg/go-udev"
)

// Lists GPUs that have a connector connected to them
func ListActiveGraphicsCards() ([]*udev.Device, error) {
	u := udev.Udev {}
	e := u.NewEnumerate()
	e.AddMatchIsInitialized()
	e.AddMatchSubsystem("drm")
	e.AddMatchProperty("DEVTYPE", "drm_connector")
	devs, err := e.Devices()
	if err != nil {
		return nil, err
	}
	var connWg sync.WaitGroup
	var connectorChan = make(chan *udev.Device, 16)

	for dev := range devs {
		device := devs[dev]
		connWg.Go(func() {
			connectStat := device.SysattrValue("status")

			var isConnected bool

			switch connectStat {
				case "connected":
					isConnected = true
				case "disconnected":
					isConnected = false
				default:
					panic("Unknown connector status: " + connectStat + " for connector: " + device.Syspath())
			}
			if isConnected {
				connectorChan <- device
			}

		})
	}
	go func () {
		connWg.Wait()
		close(connectorChan)
	} ()

	var wg sync.WaitGroup
	var cardList []*udev.Device
	var cardLock sync.Mutex

	for connector := range connectorChan {
		wg.Add(1)
		go func (conn *udev.Device) {
			defer wg.Done()
			parent := conn.Parent()
			cardLock.Lock()
			cardList = append(cardList, parent)
			cardLock.Unlock()
		} (connector)
	}
	wg.Wait()

	return cardList, nil

}