import QtQuick 2.15

Item {
    property int gay: {
        let a
        while (true) {
            if (a === undefined) {
                a = 5
            } else {
                throw "a"
            }
        }
        return a
    }
}
