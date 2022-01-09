package analysis

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func qmlPluginDump(uri []string, vmaj, vmin int) (output []byte, err error) {
	defer func() {
		if err != nil {
			log.Printf("failed to run qmlplugindump for %s %d.%d: %s", strings.Join(uri, "."), vmaj, vmin, err)
		} else {
			log.Printf("successfully ran qmlplugindump!\n%s", string(output))
		}
	}()
	log.Printf("qmltypes for %s %d.%d not found, running qmlplugindump...", strings.Join(uri, "."), vmaj, vmin)

	for _, it := range []string{"qmlplugindump", "qmlplugindump-qt5"} {
		output, err = exec.Command(it, strings.Join(uri, "."), fmt.Sprintf("%d.%d", vmaj, vmin)).Output()
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	}

	return output, nil
}
