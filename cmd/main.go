package main

import (
	"fmt"
	"github.com/djskncxm/TraceParse/pkg/core"
)

func main() {
	err := core.ReadTraceFile("../assets/code.log", func(t *core.TraceLine) {
		fmt.Println(t.Step, t.Addr, t.Instr)
	})
	if err != nil {
		fmt.Println("读取日志失败:", err)
	}
}
