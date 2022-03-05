#include <QUrl>
#include <QThread>
#include <QTimer>
#include <QJsonArray>

#include "handle.h"

const QJsonObject OK = {{"status", "ok"}};
const QJsonObject NOT_FOUND = {{"status", "notfound"}};
const QJsonObject BAD_THREAD = {{"status", "badthread"}};
const QJsonObject ASYNC = {{"status", "async"}};

void dispatchToMainThread(std::function<void()> callback)
{
    QTimer* timer = new QTimer();
    timer->moveToThread(qApp->thread());
    timer->setSingleShot(true);
    QObject::connect(timer, &QTimer::timeout, [=]()
    {
        callback();
        timer->deleteLater();
    });
    QMetaObject::invokeMethod(timer, "start", Qt::QueuedConnection, Q_ARG(int, 0));
}

QJsonObject Handle::dispatch(QJsonObject req)
{
    if (conn && QThread::currentThread() != conn->thread()) {
        dispatchToMainThread([this, req]() {
            dispatch(req);
        });

        return ASYNC;
    }

    if (req["method"] == "init") {
        init();

        return OK;
    } else if (req["method"] == "connect") {
        connect(req["target"].toString());

        return OK;
    } else if (req["method"] == "interrupt") {
        dbgClient->interrupt();

        return OK;
    } else if (req["method"] == "continue") {
        continueDebugging(req["kind"].toString());

        return OK;
    } else if (req["method"] == "eval") {
        eval(req["script"].toString());

        return OK;
    } else if (req["method"] == "backtrace") {
        dbgClient->backtrace();

        return OK;
    } else if (req["method"] == "scope") {
        dbgClient->scope(req["scope-number"].toInt());

        return OK;
    } else if (req["method"] == "frame") {
        dbgClient->frame(req["frame-number"].toInt());

        return OK;
    } else if (req["method"] == "lookup") {
        QList<int> handles;
        const auto arr = req["handles"].toArray();
        for (const auto& handle : arr) {
            handles << handle.toInt();
        }
        dbgClient->lookup(handles, req["include-source"].toBool());
        return OK;
    } else if (req["method"] == "set-breakpoint") {
        dbgClient->setBreakpoint(req["file"].toString(), req["line"].toInt());
        return OK;
    } else if (req["method"] == "set-breakpoint-enabled") {
        dbgClient->changeBreakpoint(req["number"].toInt(), req["enabled"].toBool());
        return OK;
    } else if (req["method"] == "clear-breakpoint") {
        dbgClient->clearBreakpoint(req["number"].toInt());
    }

    return NOT_FOUND;
}

void Handle::init()
{
    conn = new QQmlDebugConnection(this);
    dbgClient = new QV4DebugClient(conn);

    QObject::connect(conn, &QQmlDebugConnection::connected, [=]() {
        dbgClient->connect();
        dbgClient->setExceptionBreak(QV4DebugClient::All, true);
        notify({{"signal", "connected"}});
    });
    QObject::connect(conn, &QQmlDebugConnection::disconnected, [=]() {
        notify({{"signal", "disconnected"}});
    });
    QObject::connect(conn, &QQmlDebugConnection::socketError, [=](int err) {
        notify({{"signal", "error"}, {"error", err}});
    });
    QObject::connect(conn, &QQmlDebugConnection::socketStateChanged, [=](int state) {
        notify({{"signal", "socketStateChanged"}, {"state", state}});
    });

    QObject::connect(dbgClient, &QV4DebugClient::connected, [=]() {
        notify({{"signal", "v4-connected"}});
    });
    QObject::connect(dbgClient, &QV4DebugClient::interrupted, [=]() {
        notify({{"signal", "v4-interrupted"}});
    });
    QObject::connect(dbgClient, &QV4DebugClient::result, [=]() {
        notify({{"signal", "v4-result"}, {"command", dbgClient->response().command}, {"body", dbgClient->response().body}});
    });
    QObject::connect(dbgClient, &QV4DebugClient::failure, [=]() {
        notify({{"signal", "v4-failure"}, {"command", dbgClient->response().command}});
    });
    QObject::connect(dbgClient, &QV4DebugClient::stopped, [=]() {
        notify({{"signal", "v4-stopped"}});
    });
}

void Handle::connect(const QString& target)
{
    const auto url = QUrl::fromUserInput(target);
    conn->connectToHost(url.host(), url.port());
}

void Handle::continueDebugging(const QString& target)
{
    static const auto actions = QMap<QString, QV4DebugClient::StepAction> {
        { "continue", QV4DebugClient::Continue  },
        { "in", QV4DebugClient::In },
        { "out", QV4DebugClient::Out },
        { "next", QV4DebugClient::Next },
    };

    if (!actions.contains(target)) {
        return;
    }

    dbgClient->continueDebugging(actions[target]);
}

void Handle::eval(const QString &script)
{
    dbgClient->evaluate(script);
}
