package main

import (
	"reflect"
	"testing"
)

func str(s string) *string {
	return &s
}

func num(i int) *int {
	return &i
}

type Hm struct {
	Val int `qml:"comedy"`
}

type mu struct {
	Hms []Hm `qml:"@Hm"`
}

func TestUnmarshal(t *testing.T) {
	boolean := false
	val := Value{Boolean: str("true")}

	err := unmarshal(val, &boolean)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}

	if !boolean {
		t.Fatalf("boolean isn't true")
	}

	var it struct {
		M map[string]string `qml:"m"`
	}
	val = Value{Object: &Object{
		Name: "mald",
		Items: []Item{
			{
				Field: &Field{
					Field: "m",
					Value: Value{Map: &Map{
						Entries: []MapEntry{
							{"hi", Value{String: str("mald")}},
							{"ho", Value{String: str("mald")}},
						},
					}},
				},
			},
		},
	}}

	err = unmarshal(val, &it)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if it.M["hi"] != "mald" || it.M["ho"] != "mald" {
		t.Fatalf("wrong map values")
	}

	var arr []int
	val = Value{List: &List{
		Values: []Value{
			{Number: num(1)},
			{Number: num(2)},
		},
	}}
	err = unmarshal(val, &arr)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if !reflect.DeepEqual(arr, []int{1, 2}) {
		t.Fatalf("wrong array values: %+v vs %+v", arr, []int{1, 2})
	}

	var thonk mu
	val = Value{Object: &Object{
		Items: []Item{
			{
				Object: &Object{
					Name: "Hm",
					Items: []Item{
						{
							Field: &Field{
								Field: "comedy",
								Value: Value{Number: num(1)},
							},
						},
					},
				},
			},
			{
				Object: &Object{
					Name: "Hm",
					Items: []Item{
						{
							Field: &Field{
								Field: "comedy",
								Value: Value{Number: num(2)},
							},
						},
					},
				},
			},
		},
	}}
	err = unmarshal(val, &thonk)
	if err != nil {
		t.Fatalf("failed to unmarshal struct with children: %s", err)
	}
	if thonk.Hms[0].Val != 1 || thonk.Hms[1].Val != 2 {
		t.Fatalf("got wrong value for struct children")
	}
}
