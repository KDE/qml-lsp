package qmltypes

import "github.com/alecthomas/participle/v2"

type QMLTypesFile struct {
	Imports []Import `@@`
	Main    Object   `@@`
}

type Object struct {
	Name  string `@Ident "{"`
	Items []Item `@@* "}"`
}

type Item struct {
	Field  *Field  `(@@ |`
	Object *Object `@@) ";"?`
}

func (i *Object) FindField(s string) (Value, bool) {
	for _, it := range i.Items {
		if it.Field == nil {
			continue
		}

		if it.Field.Field == s {
			return it.Field.Value, true
		}
	}

	return Value{}, false
}

func (i *Object) ChildrenOfType(s string) []Object {
	var r []Object
	for _, it := range i.Items {
		if it.Object == nil {
			continue
		}
		if it.Object.Name == s {
			r = append(r, *it.Object)
		}
	}
	return r
}

type Field struct {
	Field string `@Ident ":"`
	Value Value  `@@`
}

type Value struct {
	Boolean        *string `@("true" | "false") |`
	List           *List   `@@ |`
	Object         *Object `@@ |`
	Map            *Map    `@@ |`
	NegativeNumber *int    `("-" @Int) |`
	Number         *int    `@Int |`
	String         *string `@String`
}

type List struct {
	Values []Value `"[" (@@ ( "," @@ )*)? "]"`
}

type Map struct {
	Entries []MapEntry `"{" (@@ ( "," @@ )* )? "}"`
}

type MapEntry struct {
	Name  string `(@Ident | @String) ":"`
	Value Value  `@@`
}

type Import struct {
	SymbolURL SymbolURL `"import" @@`
	Version   float32   `@Float`
}

type SymbolURL struct {
	Name []string `@Ident ("." @Ident)*`
}

var Parser = participle.MustBuild(&QMLTypesFile{})
