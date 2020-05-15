package main

import (
	"fmt"
	"os"
	"time"

	"container/ring"

	"github.com/dustin/go-humanize"
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/asciigraph"
	"github.com/gcla/gowid/widgets/clicktracker"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/padding"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
	asc "github.com/guptarohit/asciigraph"
	"github.com/mcuadros/go-octoprint"
	"github.com/rikkuness/octopimon/structable"
)

type status struct {
	Address string
	State   string
	Nozzle  string `label:"Nozzle Temp"`
	Bed     string `label:"Bed Temp"`
}

type job struct {
	File       string
	Progress   string
	Started    string
	Completion string
}

type PiMon struct {
	app     *gowid.App
	printer *octoprint.Client

	status status
	temps  map[string]*TemperatureGraph

	table    *structable.StructTable
	jobtable *structable.StructTable
}

func New() (*PiMon, error) {
	endpoint, epSet := os.LookupEnv("OP_ENDPOINT")
	if !epSet {
		endpoint = "http://localhost:5000"
	}

	token, tokenSet := os.LookupEnv("OP_TOKEN")
	if !tokenSet {
		return nil, fmt.Errorf("no OP_TOKEN was set")
	}

	pm := PiMon{
		status:  status{},
		temps:   make(map[string]*TemperatureGraph, 0),
		printer: octoprint.NewClient(endpoint, token),
	}

	pm.table = structable.NewStructTableWidget(pm.status)
	pm.jobtable = structable.NewStructTableWidget(job{})

	pm.temps["tool0"] = NewTempGraph("tool0")
	pm.temps["bed"] = NewTempGraph("bed")

	but := clicktracker.New(
		padding.New(
			styled.New(text.New("CANCEL"), gowid.MakePaletteRef("button")),
			gowid.VAlignMiddle{},
			gowid.RenderWithUnits{U: 1},
			gowid.HAlignMiddle{},
			gowid.RenderWithUnits{U: 6},
			padding.Options{},
		),
	)

	// TODO: This doesn't work yet, idk why
	but.OnClick(gowid.WidgetCallback{
		Name: "button",
		WidgetChangedFunction: func(app gowid.IApp, w gowid.IWidget) {
			panic("YAAY")
			// app.Run(gowid.RunFunction(func(app gowid.IApp) {
			// }))
		},
	})

	col0 := NewResizeablePile([]gowid.IContainerWidget{
		pm.temps["tool0"].GetContainer(),
		pm.temps["bed"].GetContainer(),
	})

	col1 := NewResizeablePile([]gowid.IContainerWidget{
		&gowid.ContainerWidget{
			D: gowid.RenderWithUnits{U: 6},
			IWidget: framed.New(pm.table.Widget, framed.Options{
				Title: "Status",
				Frame: framed.UnicodeFrame,
			}),
		},

		&gowid.ContainerWidget{
			D: gowid.RenderWithUnits{U: 6},
			IWidget: framed.New(pm.jobtable.Widget, framed.Options{
				Title: "Current Job",
				Frame: framed.UnicodeFrame,
			}),
		},

		&gowid.ContainerWidget{
			D:       gowid.RenderWithWeight{W: 1},
			IWidget: framed.NewUnicode(but),
		},
	})

	// Layout
	cols := NewResizeableColumns([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: col0, D: gowid.RenderWithUnits{U: 45}}, // fixed width 45
		&gowid.ContainerWidget{IWidget: col1, D: gowid.RenderWithWeight{W: 1}}, // grows to right edge
	})

	// Create App
	app, err := gowid.NewApp(gowid.AppArgs{
		View: styled.New(cols, gowid.MakePaletteRef("bg")),
		Palette: gowid.Palette{
			"bg":     gowid.MakePaletteEntry(gowid.ColorNone, gowid.ColorBlue),
			"button": gowid.MakeStyledAs(gowid.StyleBlink),
		},
	})
	if err != nil {
		return nil, err
	}
	pm.app = app

	return &pm, nil
}

func (pm *PiMon) Start() {

	// On startup get outbound IP so we know where to remote into
	if ip := GetOutboundIP(); ip != nil {
		pm.status.Address = ip.String()
	}

	// Graph data refresh thread
	go func() {
		for {
			pm.app.Run(gowid.RunFunction(func(app gowid.IApp) {

				if err := pm.checkTemperatures(); err != nil {
					return
				}

				for _, graph := range pm.temps {
					dodo := make([]float64, graph.data.Len())
					var i = 0
					graph.data.Do(func(p interface{}) {
						if p != nil {
							dodo[i] = p.(float64)
						}
						i++
					})

					graph.Widget.SetData(dodo, app)

				}
			}))

			time.Sleep(2 * time.Second)
		}
	}()

	// Table data refresh thread
	go func() {
		for {

			// Check we're still connected
			pm.app.Run(gowid.RunFunction(func(app gowid.IApp) {
				pm.checkConnection()
				pm.table.UpdateTable(app)
			}))

			switch s := octoprint.ConnectionState(pm.status.State); {

			// If offline then attemp a reconnect
			case s.IsOffline():
				go pm.reconnect()

			// If the status is printing then we should have a job to check the status of
			case s.IsPrinting():
				pm.app.Run(gowid.RunFunction(func(app gowid.IApp) {
					pm.getCurrentJobStatus()
					pm.jobtable.UpdateTable(app)
				}))

			}

			// Main loop half second tick
			time.Sleep(500 * time.Millisecond)
		}
	}()

	pm.app.SimpleMainLoop()
}

func (pm *PiMon) checkConnection() {
	r := octoprint.ConnectionRequest{}
	s, err := r.Do(pm.printer)
	if err != nil {
		pm.status.State = "Connection Error"
	} else {
		if s.Current.State.IsError() {
			pm.status.State = "Error"
		} else {
			pm.status.State = string(s.Current.State)
		}
	}

	pm.table.SetState(pm.status)
}

func (pm *PiMon) checkTemperatures() error {
	r := octoprint.StateRequest{
		Exclude: []string{"state"},
	}
	s, err := r.Do(pm.printer)
	if err != nil {
		return err
	}

	for tool, temp := range s.Temperature.Current {
		if tm, ok := pm.temps[tool]; ok {
			tm.data.Value = temp.Actual
			tm.data = tm.data.Next()
		}

	}

	pm.status.Nozzle = fmt.Sprintf("%.1f°C", s.Temperature.Current["tool0"].Actual)
	pm.status.Bed = fmt.Sprintf("%.1f°C", s.Temperature.Current["bed"].Actual)

	return nil
}

func (pm *PiMon) getCurrentJobStatus() {
	r := octoprint.JobRequest{}
	j, err := r.Do(pm.printer)
	if err != nil {
		pm.jobtable.SetState(job{}) // null out data
		return
	}

	pm.jobtable.SetState(job{
		File:       j.Job.File.Name,
		Progress:   fmt.Sprintf("%.2f%%", j.Progress.Completion),
		Completion: humanize.Time(time.Now().Add(time.Duration(j.Progress.PrintTimeLeft) * time.Second)),
		Started:    humanize.Time(time.Now().Add(-time.Duration(j.Progress.PrintTime) * time.Second)),
	})
}

func (pm *PiMon) reconnect() {
	connect := octoprint.ConnectRequest{}
	connect.Do(pm.printer)
}

type TemperatureGraph struct {
	tool   string
	data   *ring.Ring
	height int
	Widget *asciigraph.Widget
}

func NewTempGraph(tool string) *TemperatureGraph {

	graph := &TemperatureGraph{
		tool:   tool,
		data:   ring.New(35),
		height: 9,
	}

	graph.Widget = asciigraph.New([]float64{0, 1}, []asc.Option{
		asc.Height(graph.height),
		asc.Width(35),
	})

	return graph
}

func (t *TemperatureGraph) GetContainer() *gowid.ContainerWidget {
	return &gowid.ContainerWidget{
		IWidget: framed.New(t.Widget, framed.Options{
			Title: fmt.Sprintf("%s °C", t.tool),
			Frame: framed.UnicodeFrame,
		}),
		D: gowid.RenderWithUnits{U: t.height + 3},
	}
}

func main() {
	c, err := New()
	if err != nil {
		fmt.Printf("error starting: %s", err)
		return
	}

	c.Start()
}
