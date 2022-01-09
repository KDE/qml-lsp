package qmltypes_test

import (
	"qml-lsp/qmltypes"
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
	val := qmltypes.Value{Boolean: str("true")}

	err := qmltypes.Unmarshal(val, &boolean)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}

	if !boolean {
		t.Fatalf("boolean isn't true")
	}

	var it struct {
		M map[string]string `qml:"m"`
	}
	val = qmltypes.Value{Object: &qmltypes.Object{
		Name: "mald",
		Items: []qmltypes.Item{
			{
				Field: &qmltypes.Field{
					Field: "m",
					Value: qmltypes.Value{Map: &qmltypes.Map{
						Entries: []qmltypes.MapEntry{
							{"hi", qmltypes.Value{String: str("mald")}},
							{"ho", qmltypes.Value{String: str("mald")}},
						},
					}},
				},
			},
		},
	}}

	err = qmltypes.Unmarshal(val, &it)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if it.M["hi"] != "mald" || it.M["ho"] != "mald" {
		t.Fatalf("wrong map values")
	}

	var arr []int
	val = qmltypes.Value{List: &qmltypes.List{
		Values: []qmltypes.Value{
			{Number: num(1)},
			{Number: num(2)},
		},
	}}
	err = qmltypes.Unmarshal(val, &arr)
	if err != nil {
		t.Fatalf("failed to unmarshal: %s", err)
	}
	if !reflect.DeepEqual(arr, []int{1, 2}) {
		t.Fatalf("wrong array values: %+v vs %+v", arr, []int{1, 2})
	}

	var thonk mu
	val = qmltypes.Value{Object: &qmltypes.Object{
		Items: []qmltypes.Item{
			{
				Object: &qmltypes.Object{
					Name: "Hm",
					Items: []qmltypes.Item{
						{
							Field: &qmltypes.Field{
								Field: "comedy",
								Value: qmltypes.Value{Number: num(1)},
							},
						},
					},
				},
			},
			{
				Object: &qmltypes.Object{
					Name: "Hm",
					Items: []qmltypes.Item{
						{
							Field: &qmltypes.Field{
								Field: "comedy",
								Value: qmltypes.Value{Number: num(2)},
							},
						},
					},
				},
			},
		},
	}}
	err = qmltypes.Unmarshal(val, &thonk)
	if err != nil {
		t.Fatalf("failed to unmarshal struct with children: %s", err)
	}
	if thonk.Hms[0].Val != 1 || thonk.Hms[1].Val != 2 {
		t.Fatalf("got wrong value for struct children")
	}
}
