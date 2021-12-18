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
	Name       string `qml:"name"`
	ActualName string

	Exports []string `qml:"exports"`

	Enums      []Enum     `qml:"@Enum"`
	Properties []Property `qml:"@Property"`
}

func (c *Component) GetActualName() {
	if len(c.Exports) < 1 {
		c.ActualName = c.Name
		return
	}
	v := c.Exports[0]
	slash := strings.Index(v, "/")
	space := strings.Index(v, " ")
	if slash == -1 || space == -1 {
		c.ActualName = c.Name
		return
	}
	slash++
	c.ActualName = v[slash:space]
}

type Enum struct {
	Name   string         `qml:"name"`
	Values map[string]int `qml:"?values"`
}

type Property struct {
	Name string `qml:"name"`
	Type string `qml:"type"`
}
