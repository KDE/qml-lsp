import QtQuick 2.15

Item {
    property int gay: {
        let a = null
        while (a === null) {
            if (b === 10) {
                break
            }
            a = 3
        }
        return a
    }
}
