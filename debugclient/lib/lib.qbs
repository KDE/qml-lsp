DynamicLibrary {
    name: "qmldap"
    files: ["*.cpp", "*.h"]

    install: true
    installDir: "lib64"

    Depends { name: "Qt"; submodules: ["qml", "qml-private", "qmldebug-private"] }
}