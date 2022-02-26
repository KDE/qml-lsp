import org.kde.kirigami 2.12 as Kirigami

Kirigami.Heading {
    kili: 2
    level: 2

    Namako.Telo.nasa: 5

    readonly property int yourMom: 5
    readonly property Kirigami.Mom yourMom: 5

    enum Weird {
        A,
        B,
        C
    }

    // a
    component Hello : World {
    }

    namako: Kirigami.AboutPage {
        namako: {
            let telo = 1
        }
    }
}
