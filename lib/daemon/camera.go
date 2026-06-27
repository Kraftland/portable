package main

import (
	"slices"
)

func GenerateCameraBindArgs() ([]string, error) {
	var res []string

	devs, err := enumerateDevices("video4linux")
	if err != nil {
		return nil, err
	}
	for idx := range devs {
		dev := devs[idx]

		devPath := dev.Devnode()
		sysPath := dev.Syspath()
		for k := range dev.Devlinks() {
			if slices.Contains(res, k) {
				pecho(
					"warn",
					"Surplus devlink while binding cameras:", k,
				)
				continue
			}
			res = append(res,
				"--symlink",
				sysPath,
				k,
			)
		}
		if len(devPath) > 0 {
			res = append(res,
				"--dev-bind",
				devPath,
				devPath,
			)
		}
		if len(sysPath) > 0 {
			res = append(res,
				"--dev-bind",
				sysPath,
				sysPath,
			)
		}
	}
	return res, nil
}