package main

import "strings"

type Module struct {
	Dependencies []string    `qml:"dependency"`
	Components   []Component `qml:"@Component"`
}

func saneify(s string) string {
	// org.kde.kirigami/Heading 2.0
	if strings.Contains(s, "/") && strings.Contains(s, " ") {
		return strings.Split(strings.Split(s, "/")[1], " ")[0]
	}

	return s
}

type Component struct {
	Name string `qml:"name"`

	Enums      []Enum     `qml:"@Enum"`
	Properties []Property `qml:"@Property"`
}

type Enum struct {
	Name   string         `qml:"name"`
	Values map[string]int `qml:"values"`
}

type Property struct {
	Name string `qml:"name"`
	Type string `qml:"type"`
}
