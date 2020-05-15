package structable

import (
	"reflect"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/fill"
	"github.com/gcla/gowid/widgets/table"
)

func structToTableData(input interface{}) [][]string {
	typ := reflect.TypeOf(input)
	val := reflect.ValueOf(input)

	data := make([][]string, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		label := typ.Field(i).Tag.Get("label")
		if label == "" {
			label = typ.Field(i).Name
		}

		value := val.Field(i).String()
		if value == "" {
			value = "-"
		}

		data[i] = []string{label, value}
	}

	return data
}

type StructTable struct {
	state  interface{}
	model  *table.SimpleModel
	Widget *table.Widget
}

func NewStructTableWidget(initial interface{}) *StructTable {

	// Field the longest field name to collapse the first col
	var longest = 0
	typ := reflect.TypeOf(initial)
	for i := 0; i < typ.NumField(); i++ {
		label := typ.Field(i).Tag.Get("label")
		if label == "" {
			label = typ.Field(i).Name
		}

		nameLength := len(label)
		if nameLength > longest {
			longest = nameLength
		}
	}

	t := table.NewSimpleModel([]string{}, structToTableData(initial), table.SimpleOptions{
		Layout: table.LayoutOptions{
			Widths: []gowid.IWidgetDimension{
				gowid.RenderWithUnits{U: longest + 1},
			},
		},
		Style: table.StyleOptions{
			VerticalSeparator: fill.New(' '),
			CellStyleProvided: true,
		},
	})

	return &StructTable{
		state:  initial,
		model:  t,
		Widget: table.New(t, table.Options{}),
	}
}

func (t *StructTable) UpdateTable(app gowid.IApp) {
	t.model.Data = structToTableData(t.state)
	t.Widget.SetModel(t.model, app)
}

func (t *StructTable) SetState(state interface{}) {
	t.state = state
}
