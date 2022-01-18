#pragma once

#include <QCoreApplication>
#include <QJsonObject>
#include <QJsonDocument>

#include <private/qqmldebugconnection_p.h>
#include <private/qv4debugclient_p.h>

#include "lib.h"

struct Handle : public QObject {

    Q_OBJECT

public:
    QCoreApplication* app = nullptr;
    JSONNotification callback;
    void* userData = nullptr;
    QByteArray lastRet;

    QQmlDebugConnection* conn = nullptr;
    QV4DebugClient* dbgClient = nullptr;

    QJsonObject dispatch(QJsonObject);
    void notify(QJsonObject obj)
    {
        lastRet = QJsonDocument(obj).toJson(QJsonDocument::Compact);
        callback(userData, lastRet.constData());
    }

    void init();
    void connect(const QString& url);
    void continueDebugging(const QString& action);
    void eval(const QString& script);
};
