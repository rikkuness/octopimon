package main

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/pile"
)

type ResizeableColumnsWidget struct {
	*columns.Widget
	offset int
}

func NewResizeableColumns(widgets []gowid.IContainerWidget) *ResizeableColumnsWidget {
	res := &ResizeableColumnsWidget{}
	res.Widget = columns.New(widgets)
	return res
}

type ResizeablePileWidget struct {
	*pile.Widget
	offset int
}

func NewResizeablePile(widgets []gowid.IContainerWidget) *ResizeablePileWidget {
	res := &ResizeablePileWidget{}
	res.Widget = pile.New(widgets)
	return res
}
