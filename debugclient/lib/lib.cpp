#include <QJsonDocument>

#include "lib.h"
#include "handle.h"

char** argv = {nullptr};
int argc = 0;

Handle* makeLibraryHandle(void* userData, void* callback)
{
    auto handle = new Handle;
    handle->userData = userData;
    handle->callback = (JSONNotification)callback;
    handle->app = new QCoreApplication(argc, argv);

    return handle;
}

void execHandle(Handle* handle)
{
    handle->app->exec();
}

const char* invokeHandle(Handle* handle, const char* op)
{
    auto arr = QByteArray(op);

    auto res = handle->dispatch(QJsonDocument::fromJson(arr).object());
    handle->lastRet = QJsonDocument(res).toJson(QJsonDocument::Compact);

    return handle->lastRet.constData();
}

void freeLibraryHandle(Handle* handle)
{
    delete handle;
}
