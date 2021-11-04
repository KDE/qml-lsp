package libpaths

import (
	"os"
	"os/exec"
	"strings"
)

func find(b []byte) string {
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, "QT_INSTALL_QML:") {
			return strings.TrimPrefix(ln, "QT_INSTALL_QML:")
		}
	}
	return ""
}

func get() (string, error) {
	var err error
	var output []byte
	for _, it := range []string{"qmake", "qmake-qt5"} {
		output, err = exec.Command(it, "-query").Output()
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return "", err
	}
	return find(output), err
}

func Paths() []string {
	env := strings.FieldsFunc(os.Getenv("QML2_IMPORT_PATH"), func(r rune) bool {
		return r == ':'
	})
	qt, err := get()
	if err != nil {
		panic("failed to get qt: " + err.Error())
	}

	return append(env, qt)
}
