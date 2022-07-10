import QtQuick 2.15

Item {
    property int gay: {
        let a = null
        while (a === null) {
            a = 3
        }
        return a
    }
}
