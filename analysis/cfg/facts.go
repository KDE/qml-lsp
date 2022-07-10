package cfg

type Facts struct {
	currentFactID int

	Graph *Graph
	Facts map[EdgeID][]Fact
}

func NewFacts(g *Graph) *Facts {
	return &Facts{
		currentFactID: 0,
		Graph:         g,
		Facts:         map[EdgeID][]Fact{},
	}
}
func (f *Facts) CurrentFactID() int {
	return f.currentFactID
}
func (f *Facts) NextFactID() int {
	f.currentFactID++
	return f.currentFactID
}
func (f *Facts) Record(fact Fact, forEdge EdgeID) {
	if v, ok := f.Facts[forEdge]; ok {
		f.Facts[forEdge] = append(v, fact)
	} else {
		f.Facts[forEdge] = []Fact{fact}
	}
}

type Fact interface {
	FactID() int
	// true -> this fact must be true
	// false -> this fact may be false
	Must() bool
	String() string
	Hash() string
}
