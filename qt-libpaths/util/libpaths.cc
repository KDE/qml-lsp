#include <QtCore/qlibraryinfo.h>
#include "libpaths.h"

static std::string it = QLibraryInfo::location(QLibraryInfo::Qml2ImportsPath).toStdString();

const char* getLibraryPaths()
{
    return it.c_str();
}
