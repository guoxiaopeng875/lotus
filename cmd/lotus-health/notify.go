package main

import (
	"fmt"
	"os"
)

func notifyHandler(n string, ch chan interface{}, sCh chan os.Signal) (string, error) {
	select {
	// alerts to restart systemd unit
	case <-ch:
		fmt.Println("restart lotus daemon")
		return "", nil
		//statusCh := make(chan string, 1)
		//c, err := dbus.New()
		//if err != nil {
		//	return "", err
		//}
		//_, err = c.TryRestartUnit(n, "fail", statusCh)
		//if err != nil {
		//	return "", err
		//}
		//select {
		//case result := <-statusCh:
		//	return result, nil
		//}
	// SIGTERM
	case <-sCh:
		os.Exit(1)
		return "", nil
	}
}
