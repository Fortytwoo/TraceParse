package core

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// TraceLine 表示日志中的一条指令快照
type TraceLine struct {
	Step   uint32
	Addr   uint64
	Offset uint64
	Instr  string
	Regs   [31]uint64 // x0-x30
	SP     uint64
	PC     uint64
}

// TraceManager 管理指令跟踪
type TraceManager struct {
	Instructions []*TraceLine
	PrevLine     *TraceLine // 添加上一条指令的缓存
	CurrentIndex int
	totalLines   int // 文件总行数（可能大于Instructions长度）
	loadedRange  [2]int // 已加载的范围[start, end)
}

func NewTraceManager() *TraceManager {
	return &TraceManager{
		Instructions: make([]*TraceLine, 0),
		CurrentIndex: 0,
		totalLines:   0,
		loadedRange:  [2]int{-1, -1},
	}
}

func (tm *TraceManager) GetCurrent() *TraceLine {
	if tm.CurrentIndex < 0 || tm.CurrentIndex >= len(tm.Instructions) {
		return nil
	}
	return tm.Instructions[tm.CurrentIndex]
}

func (tm *TraceManager) GetLine(index int) *TraceLine {
	if index >= 0 && index < len(tm.Instructions) {
		return tm.Instructions[index]
	}
	return nil
}

func (tm *TraceManager) Total() int {
	return tm.totalLines
}

// ParseLine 解析日志中的一行
func ParseLine(line string) (*TraceLine, error) {
	fields := strings.Split(line, "|")
	if len(fields) != 37 {
		return nil, fmt.Errorf("字段数量不对: %d", len(fields))
	}

	t := &TraceLine{}

	// step
	step, err := strconv.ParseUint(strings.TrimSpace(fields[0]), 16, 32)
	if err != nil {
		return nil, fmt.Errorf("解析 step 失败: %v", err)
	}
	t.Step = uint32(step)

	// addr
	addr, err := strconv.ParseUint(strings.TrimSpace(fields[1]), 0, 64)
	if err != nil {
		return nil, fmt.Errorf("解析 addr 失败: %v", err)
	}
	t.Addr = addr

	// offset
	offset, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 0, 64)
	if err != nil {
		return nil, fmt.Errorf("解析 offset 失败: %v", err)
	}
	t.Offset = offset

	// instr
	t.Instr = strings.TrimSpace(fields[3])
	if strings.HasPrefix(t.Instr, "\"") && strings.HasSuffix(t.Instr, "\"") {
		t.Instr = t.Instr[1 : len(t.Instr)-1]
	}

	// x0-x28
	for i := 0; i <= 28; i++ {
		val, err := strconv.ParseUint(strings.TrimSpace(fields[4+i]), 0, 64)
		if err != nil {
			return nil, fmt.Errorf("解析 x%d 失败: %v", i, err)
		}
		t.Regs[i] = val
	}

	// x29
	val, err := strconv.ParseUint(strings.TrimSpace(fields[33]), 0, 64)
	if err != nil {
		return nil, fmt.Errorf("解析 x29 失败: %v", err)
	}
	t.Regs[29] = val

	// x30
	val, err = strconv.ParseUint(strings.TrimSpace(fields[34]), 0, 64)
	if err != nil {
		return nil, fmt.Errorf("解析 x30 失败: %v", err)
	}
	t.Regs[30] = val

	// sp
	val, err = strconv.ParseUint(strings.TrimSpace(fields[35]), 0, 64)
	if err != nil {
		return nil, fmt.Errorf("解析 sp 失败: %v", err)
	}
	t.SP = val

	// pc
	val, err = strconv.ParseUint(strings.TrimSpace(fields[36]), 0, 64)
	if err != nil {
		return nil, fmt.Errorf("解析 pc 失败: %v", err)
	}
	t.PC = val

	return t, nil
}

// 流式读取日志文件，但只加载一部分
func ReadTraceFile(filename string, tm *TraceManager) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 首先，统计总行数并扫描行位置
	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}
	tm.totalLines = lineCount
	
	// 重置文件指针
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	
	// 加载初始窗口（当前行附近的窗口）
	windowSize := 2000 // 加载2000行，足够显示
	start := 0
	if tm.CurrentIndex > windowSize/2 {
		start = tm.CurrentIndex - windowSize/2
		if start < 0 {
			start = 0
		}
	}
	
	end := start + windowSize
	if end > tm.totalLines {
		end = tm.totalLines
		start = end - windowSize
		if start < 0 {
			start = 0
		}
	}
	
	// 记录加载范围
	tm.loadedRange = [2]int{start, end}
	
	// 清空现有指令
	tm.Instructions = make([]*TraceLine, 0)
	
	// 扫描并加载指定范围的行
	currentLine := 0
	for scanner.Scan() {
		if currentLine >= start && currentLine < end {
			line := scanner.Text()
			traceLine, err := ParseLine(line)
			if err != nil {
				fmt.Printf("解析错误 第%d行: %v\n", currentLine+1, err)
				continue
			}
			tm.Instructions = append(tm.Instructions, traceLine)
		}
		currentLine++
		
		// 如果已经过了end，就停止
		if currentLine >= end {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (tm *TraceManager) GetPrevLine() *TraceLine {
	if tm.CurrentIndex <= 0 || tm.CurrentIndex >= len(tm.Instructions) {
		return nil
	}
	return tm.Instructions[tm.CurrentIndex-1]
}

// 在 Next 和 Prev 方法中更新 PrevLine 缓存
func (tm *TraceManager) Next() bool {
	if tm.CurrentIndex < len(tm.Instructions)-1 {
		tm.PrevLine = tm.GetCurrent() // 缓存当前指令作为下一次的上一条
		tm.CurrentIndex++
		return true
	}
	return false
}

func (tm *TraceManager) Prev() bool {
	if tm.CurrentIndex > 0 {
		tm.CurrentIndex--
		// 更新 PrevLine，现在上一条是索引-2
		if tm.CurrentIndex-1 >= 0 {
			tm.PrevLine = tm.Instructions[tm.CurrentIndex-1]
		} else {
			tm.PrevLine = nil
		}
		return true
	}
	return false
}

func (tm *TraceManager) GoTo(index int) bool {
	if index >= 0 && index < tm.totalLines {
		// 检查是否需要重新加载窗口
		if index < tm.loadedRange[0] || index >= tm.loadedRange[1] {
			// 需要重新加载窗口
			// 在实际实现中，这里应该触发异步重新加载
			// 暂时先更新索引
		}
		// 更新 PrevLine
		if index-1 >= 0 {
			tm.PrevLine = tm.GetLine(index - 1)
		} else {
			tm.PrevLine = nil
		}
		tm.CurrentIndex = index
		return true
	}
	return false
}

// 修改 AddInstruction 方法
func (tm *TraceManager) AddInstruction(t *TraceLine) {
	tm.Instructions = append(tm.Instructions, t)
	tm.totalLines = len(tm.Instructions)
}

