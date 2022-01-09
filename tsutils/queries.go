package tsutils

import (
	qml "qml-lsp/treesitter-qml"
	"reflect"

	sitter "github.com/smacker/go-tree-sitter"
)

func InitQueriesStructure(q interface{}) error {
	v := reflect.ValueOf(q)
	strct := v.Elem().Type()
	for i := 0; i < strct.NumField(); i++ {
		k := strct.Field(i)
		query, err := sitter.NewQuery([]byte(k.Tag), qml.GetLanguage())
		if err != nil {
			return err
		}
		v.Elem().Field(i).Set(reflect.ValueOf(query))
	}
	return nil
}
