package util

import (
	"bytes"
	"os/exec"
)

func JournalctlPrint(s string) {
	cmd := exec.Command("systemd-cat")
	var b bytes.Buffer
	b.Write([]byte(s))
	cmd.Stdin = &b
	err := cmd.Start()

	if err != nil {
		panic(err)
	}

	err = cmd.Wait()
	if err != nil {
		panic(err)
	}
}
