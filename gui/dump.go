package gui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	g "github.com/AllenDang/giu"
	"github.com/andrewpmartinez/grid/dump"
	"github.com/sirupsen/logrus"
)

type DumpWindow struct {
	Dump         *dump.Dump
	masterWindow *g.MasterWindow
	logger       dump.Logger

	buildFunctionRowsOnce sync.Once
	functionRows          []*g.TableRowWidget
	path                  string
	editor                *g.CodeEditorWidget
	routineText           string
}

func NewDumpWindow(logger dump.Logger) *DumpWindow {
	editor := g.CodeEditor().
		ShowWhitespaces(false).
		TabSize(2).
		Border(true)

	return &DumpWindow{
		buildFunctionRowsOnce: sync.Once{},
		editor:                editor,
		logger:                logger,
	}
}

func (dumpWindow *DumpWindow) LoadFile(path string) {
	var err error
	dumpWindow.path = path
	dumpWindow.Dump, err = dump.ParseFile(path, nil)

	if err != nil {
		errStr := fmt.Sprintf("error parsing file [%s]: %v", path, err)
		logrus.Error(errStr)
	}

}

func (dumpWindow *DumpWindow) buildRows() []*g.TableRowWidget {
	dumpWindow.buildFunctionRowsOnce.Do(func() {

		maxFuncName := ""
		maxCount := 0
		dumpWindow.functionRows = make([]*g.TableRowWidget, dumpWindow.Dump.Stats.RoutinesByFunction.Len())

		type RoutineStats struct {
			Total  int
			Widget *g.TableRowWidget
		}
		statsSlice := make([]RoutineStats, dumpWindow.Dump.Stats.RoutinesByFunction.Len())

		i := 0
		for el := dumpWindow.Dump.Stats.RoutinesByFunction.Front(); el != nil; el = el.Next() {
			routineStats, _ := el.Value.(*dump.RoutineStats)

			funcName := el.Key.(string)
			numTotalRoutines := len(routineStats.Routines)
			numUniqueRoutines := len(routineStats.RoutinesBySignature)

			if len(funcName) > len(maxFuncName) {
				maxFuncName = funcName
			}

			if numTotalRoutines > maxCount {
				maxCount = numTotalRoutines
			}

			button := g.Button("Open").OnClick(func() {
				dumpWindow.OpenFunctionDetail(funcName)
			})

			numTotalRoutinesLabel := g.Label(strconv.Itoa(numTotalRoutines))
			numUniqueRoutinesLabel := g.Label(strconv.Itoa(numUniqueRoutines))

			funcLabel := g.Label(funcName)

			widget := g.TableRow(
				button,
				numTotalRoutinesLabel,
				numUniqueRoutinesLabel,
				funcLabel,
			)

			statsSlice[i] = RoutineStats{
				Total:  numTotalRoutines,
				Widget: widget,
			}

			i++
		}

		sort.Slice(statsSlice, func(i, j int) bool {
			return statsSlice[i].Total > statsSlice[j].Total
		})

		for j, stats := range statsSlice {
			dumpWindow.logger.Infof("Slice sorted: %d #%d", stats.Total, j)
			dumpWindow.functionRows[j] = stats.Widget
		}

		//		masterWindow.functionRows[0].BgColor(&(color.RGBA{200, 100, 100, 255}))
	})

	return dumpWindow.functionRows
}

func (dumpWindow *DumpWindow) loop() {
	masterWidth, masterHeight := dumpWindow.masterWindow.GetSize()

	g.Window("DumpStatus").
		Flags(g.WindowFlagsNoResize|g.WindowFlagsNoCollapse|g.WindowFlagsNoMove|g.WindowFlagsNoTitleBar).
		Size(float32(masterWidth), 30).
		Layout(
			g.Label("File: " + dumpWindow.path),
		)

	g.Window("DumpNav").
		Flags(g.WindowFlagsNoResize|g.WindowFlagsNoCollapse|g.WindowFlagsNoMove|g.WindowFlagsNoTitleBar).
		Pos(0, 31).
		Size(650, float32(math.Max(float64(masterHeight-31), 50))).
		Layout(
			g.Table().
				Columns(
					g.TableColumn("").Flags(g.TableColumnFlagsWidthFixed).InnerWidthOrWeight(50),
					g.TableColumn("Total").Flags(g.TableColumnFlagsWidthFixed).InnerWidthOrWeight(70),
					g.TableColumn("Unique").Flags(g.TableColumnFlagsWidthFixed).InnerWidthOrWeight(70),
					g.TableColumn("Function"),
				).
				Freeze(0, 1).
				FastMode(true).
				Rows(dumpWindow.buildRows()...),
		)

	g.Window("RoutineDetails").
		Flags(g.WindowFlagsNoResize|g.WindowFlagsNoCollapse|g.WindowFlagsNoMove|g.WindowFlagsNoTitleBar).
		Pos(650, 31).
		Size(-1, float32(math.Max(float64(masterHeight-31), 50))).
		Layout(
			g.InputTextMultiline(&dumpWindow.routineText).
				Size(float32(math.Max(float64(masterWidth-650), 400)), float32(math.Max(float64(masterHeight-31), 50))),
		)
}

func (dumpWindow *DumpWindow) Run() {
	dumpWindow.masterWindow = g.NewMasterWindow("Dump", 1700, 800, 0)
	dumpWindow.masterWindow.Run(dumpWindow.loop)
}

func (dumpWindow *DumpWindow) OpenFunctionDetail(funcName string) {
	routineStats := dumpWindow.Dump.Stats.GetRoutinesByFunction(funcName)

	// Sort by occurrences
	type RoutineStats struct {
		Total     int
		Signature string
		Routines  []*dump.Routine
	}
	routinesSlice := make([]RoutineStats, 0, len(routineStats.RoutinesBySignature))
	for signature, routines := range routineStats.RoutinesBySignature {
		routinesSlice = append(routinesSlice, RoutineStats{
			Total:     len(routines),
			Signature: signature,
			Routines:  routines,
		})
	}

	sort.Slice(routinesSlice, func(i, j int) bool {
		return routinesSlice[i].Total > routinesSlice[j].Total
	})

	builder := strings.Builder{}
	start := time.Now()
	for _, stats := range routinesSlice {
		signature := stats.Signature
		routines := stats.Routines
		builder.Write([]byte(fmt.Sprintf("Signature: %s\nOccurences: %d \n\n", signature, stats.Total)))

		builder.Write([]byte(routines[0].Raw()))
		builder.Write([]byte("\n"))
		builder.Write([]byte("go routine ids: "))
		isFirst := true
		idsPerLineCount := 0
		for _, routine := range routines {
			if !isFirst {
				builder.Write([]byte(","))
				if idsPerLineCount > 50 {
					idsPerLineCount = 0
					builder.Write([]byte("\n"))
				}
			} else {
				isFirst = false
			}
			builder.Write([]byte(strconv.Itoa(routine.Id)))
			idsPerLineCount++
		}
		builder.Write([]byte("\n\n--------------------------------------------------------------------------------------------------------------------\n\n"))

	}

	dumpWindow.routineText = builder.String()

	duration := time.Now().Sub(start)
	println(duration.String())
}
