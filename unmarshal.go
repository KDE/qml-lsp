package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func unmarshalI(v Value, valOf reflect.Value) error {
	typOf := valOf.Type()

	switch typOf.Elem().Kind() {
	case reflect.Slice:
		if v.List == nil {
			return errors.New("not an slice")
		}
		arrElmType := valOf.Elem().Type().Elem()
		theSlice := reflect.New(reflect.SliceOf(arrElmType)).Elem()
		for _, it := range v.List.Values {
			nova := reflect.New(arrElmType)
			err := unmarshalI(it, nova)
			if err != nil {
				return fmt.Errorf("failed to unmarshal slice element: %+w", err)
			}
			theSlice = reflect.Append(theSlice, nova.Elem())
		}
		valOf.Elem().Set(theSlice)
	case reflect.Struct:
		if v.Object == nil {
			return errors.New("not an object")
		}
		for i := 0; i < valOf.Elem().NumField(); i++ {
			f := valOf.Elem().Field(i)
			ft := valOf.Elem().Type().Field(i)

			name, ok := ft.Tag.Lookup("qml")
			if !ok {
				name = ft.Name
			}

			if strings.HasPrefix(name, "@") {
				if ft.Type.Kind() != reflect.Slice {
					return fmt.Errorf("got a children thing, but field type isn't slice")
				}

				arrElmType := ft.Type.Elem()
				theSlice := reflect.New(reflect.SliceOf(arrElmType)).Elem()
				items := v.Object.ChildrenOfType(name[1:])
				for _, it := range items {
					nova := reflect.New(arrElmType)
					err := unmarshalI(Value{Object: &it}, nova)
					if err != nil {
						return fmt.Errorf("failed to unmarshal slice element: %+w", err)
					}
					theSlice = reflect.Append(theSlice, nova.Elem())
				}

				f.Set(theSlice)
			} else {
				fV, ok := v.Object.FindField(name)
				if !ok {
					continue
				}

				err := unmarshalI(fV, f.Addr())
				if err != nil {
					return fmt.Errorf("failed to unmarshal struct field: %+w", err)
				}
			}

		}
	case reflect.Map:
		if v.Map == nil {
			return errors.New("not a map")
		}

		keyKind := typOf.Elem().Key()
		if keyKind.Kind() != reflect.String {
			return errors.New("map key isn't string")
		}

		newMap := reflect.MakeMapWithSize(typOf.Elem(), 0)

		valKind := typOf.Elem().Elem()

		for _, it := range v.Map.Entries {
			new := reflect.New(valKind)
			err := unmarshalI(it.Value, new)
			if err != nil {
				return fmt.Errorf("failed to unmarshal map value: %+w", err)
			}
			newMap.SetMapIndex(reflect.ValueOf(strings.Trim(it.Name, `"`)), new.Elem())
		}

		valOf.Elem().Set(newMap)
	case reflect.Bool:
		if v.Boolean == nil {
			return errors.New("not a boolean")
		}
		valOf.Elem().SetBool(*v.Boolean == "true")
	case reflect.Int:
		if v.Number == nil && v.NegativeNumber == nil {
			return errors.New("not an integer")
		}
		if v.Number != nil {
			valOf.Elem().SetInt(int64(*v.Number))
		} else {
			valOf.Elem().SetInt(-int64(*v.NegativeNumber))
		}
	case reflect.String:
		if v.String == nil {
			return errors.New("not a string")
		}
		valOf.Elem().SetString(strings.Trim(*v.String, `"`))
	default:
		panic("unhandled " + typOf.Elem().Kind().String())
	}

	return nil
}

func unmarshal(v Value, i interface{}) error {
	valOf := reflect.ValueOf(i)
	typOf := reflect.TypeOf(i)

	if typOf.Kind() != reflect.Ptr {
		return errors.New("input value not a pointer")
	}

	return unmarshalI(v, valOf)
}
