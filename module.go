package main

type Module struct {
	Dependencies []string    `qml:"dependency"`
	Components   []Component `qml:"@Component"`
}

type Component struct {
	Name string `qml:"name"`

	Enums []Enum `qml:"@Enum"`
}

type Enum struct {
	Name   string         `qml:"name"`
	Values map[string]int `qml:"values"`
}
