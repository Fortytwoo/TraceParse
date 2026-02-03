package main

import (
	"fmt"
	"github.com/djskncxm/TraceParse/pkg/tui"
	"github.com/rivo/tview"
	"time"
)

func main() {
	app := tview.NewApplication()

	top := tui.NewBlock("汇编", true)
	middle := tui.NewBlock("寄存器", true)
	bottom := tui.NewBlock("用户交互", false)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(top, 0, 1, false).
		AddItem(middle, 0, 1, false).
		AddItem(bottom, 0, 1, false)

	tvTOP := top.GetItem(0).(*tview.TextView)

	counter := 0
	tui.DynamicUpdate(app, tvTOP, func() string {
		counter++
		return fmt.Sprintf("%d", counter)
	}, 50*time.Millisecond)

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
