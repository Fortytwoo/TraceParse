package main

import (
	"fmt"
	"github.com/djskncxm/TraceParse/pkg/core"
	"github.com/djskncxm/TraceParse/pkg/tui"
	"github.com/rivo/tview"
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

	// 创建一个channel来传递指令
	instructionChan := make(chan string, 1000) // 缓冲通道，避免阻塞

	go core.LoadInstructions("../assets/code.log", instructionChan)
	go func() {
		lines := []string{}
		current := 0
		const panelHeight = 16

		for line := range instructionChan {
			lines = append(lines, line)

			app.QueueUpdateDraw(func() {
				tvTOP.Clear()

				// 计算显示窗口
				start := current - panelHeight/2
				if start < 0 {
					start = 0
				}
				end := start + panelHeight
				if end > len(lines) {
					end = len(lines)
					start = end - panelHeight
					if start < 0 {
						start = 0
					}
				}

				// 输出窗口内容，高亮当前行
				for i := start; i < end; i++ {
					if i == current {
						fmt.Fprintf(tvTOP, "[yellow]> %s[white]\n", lines[i])
					} else {
						fmt.Fprintf(tvTOP, "  %s\n", lines[i])
					}
				}
			})
			current++
		}

		// 所有指令显示完成
		app.QueueUpdateDraw(func() {
			fmt.Fprintf(tvTOP, "[green]所有指令已显示完成！\n")
		})
	}()

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
