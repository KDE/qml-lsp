package libpaths

/*
#cgo LDFLAGS: -Lutil/_build -lpaths
#include "util/libpaths.h"
*/
import "C"
import (
	"os"
	"strings"
)

func Paths() []string {
	env := strings.FieldsFunc(os.Getenv("QML2_IMPORT_PATH"), func(r rune) bool {
		return r == ':'
	})
	qt := C.GoString(C.getLibraryPaths())

	return append(env, qt)
}
