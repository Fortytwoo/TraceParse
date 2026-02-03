package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NewBlock 创建一个带文本的模块，可选分割线
func NewBlock(text string, drawLine bool) *tview.Flex {
	// 文本区
	tv := tview.NewTextView().
		SetText(text).
		SetTextColor(tcell.ColorDefault).
		SetDynamicColors(true)

	if !drawLine {
		return tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tv, 0, 1, false)
	}

	line := tview.NewBox().
		SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
			for i := 0; i < width; i++ {
				screen.SetContent(x+i, y, '─', nil, tcell.StyleDefault.Foreground(tcell.ColorWhite))
			}
			return x, y, width, 1
		})

	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tv, 0, 1, false).
		AddItem(line, 1, 0, false)
}
