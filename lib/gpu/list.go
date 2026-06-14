package gpu

import (
	"path/filepath"
	"strings"
	"sync"

	udev "github.com/jochenvg/go-udev"
)

// Lists available graphics card, returns a list of card nodes
func ListGraphicsCard () ([]*udev.Device, error) {
	u := udev.Udev {}
	e := u.NewEnumerate()
	e.AddMatchIsInitialized()
	e.AddMatchSubsystem("drm")
	e.AddMatchProperty("DEVTYPE", "drm_minor")
	devs, err := e.Devices()
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	var cards []*udev.Device
	var cardsLock sync.Mutex

	for dev := range devs {
		device := devs[dev]
		wg.Go(func() {
			nodeFile := filepath.Base(device.Devnode())
			if strings.HasPrefix(nodeFile, "render") {
				return
			}
			cardsLock.Lock()
			cards = append(cards, device)
			cardsLock.Unlock()
		})
	}
	wg.Wait()
	return cards, nil
}