package main

import (
	"encoding/json"
	"runtime"
)

type obj map[string]interface{}

func serverMain(h *handle) {
	StartServer(h)
}

// func pront(s string) {
// 	cmd := exec.Command("systemd-cat")
// 	var b bytes.Buffer
// 	b.Write([]byte(s))
// 	cmd.Stdin = &b
// 	err := cmd.Start()

// 	if err != nil {
// 		panic(err)
// 	}

// 	err = cmd.Wait()
// 	if err != nil {
// 		panic(err)
// 	}
// }

func main() {
	h := new(handle)
	h.init()
	h.Callback = func(i json.RawMessage) {
		println(string(i))
	}

	initted := make(chan bool)

	go func() {
		<-initted

		runtime.LockOSThread()

		serverMain(h)
	}()

	runtime.LockOSThread()
	h.invoke(obj{"method": "init"})
	initted <- true
	h.exec()
}
