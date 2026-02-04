package core

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

// FileCache 文件缓存管理器
type FileCache struct {
	filename       string
	linePositions  []int64
	cache          map[int]*TraceLine
	cacheMutex     sync.RWMutex
	totalLines     int
	cacheSize      int
	prefetchWindow int
	prefetchMutex  sync.Mutex
	prefetchQueue  chan int
	stopPrefetch   chan bool
}

// NewFileCache 创建新的文件缓存
func NewFileCache(filename string, cacheSize int) (*FileCache, error) {
	cache := &FileCache{
		filename:       filename,
		cache:          make(map[int]*TraceLine),
		linePositions:  make([]int64, 0),
		cacheSize:      cacheSize,
		prefetchWindow: 200, // 预加载窗口大小
		prefetchQueue:  make(chan int, 100),
		stopPrefetch:   make(chan bool, 1),
	}
	
	// 扫描文件获取行位置和总行数
	if err := cache.scanFile(); err != nil {
		return nil, err
	}
	
	// 启动预加载协程
	go cache.prefetchWorker()
	
	// 预加载前几行
	cache.prefetchAround(0)
	
	return cache, nil
}

// scanFile 扫描文件获取每行的起始位置
func (fc *FileCache) scanFile() error {
	file, err := os.Open(fc.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var pos int64 = 0
	
	fc.linePositions = make([]int64, 0)
	fc.totalLines = 0
	
	for scanner.Scan() {
		fc.linePositions = append(fc.linePositions, pos)
		pos += int64(len(scanner.Bytes()) + 1) // +1 for newline
		fc.totalLines++
		
		// 可选：每扫描一定行数输出进度
		if fc.totalLines%100000 == 0 {
			fmt.Printf("扫描进度: %d 行\n", fc.totalLines)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return err
	}
	
	fmt.Printf("文件扫描完成: 总行数 = %d\n", fc.totalLines)
	return nil
}

// GetLine 获取指定行的指令
func (fc *FileCache) GetLine(index int) *TraceLine {
	if index < 0 || index >= fc.totalLines {
		return nil
	}
	
	// 从缓存获取
	fc.cacheMutex.RLock()
	if line, exists := fc.cache[index]; exists {
		fc.cacheMutex.RUnlock()
		
		// 触发异步预加载
		select {
		case fc.prefetchQueue <- index:
		default:
			// 如果队列满，跳过
		}
		
		return line
	}
	fc.cacheMutex.RUnlock()
	
	// 从文件加载
	line := fc.loadLineFromFile(index)
	if line == nil {
		return nil
	}
	
	// 添加到缓存
	fc.cacheMutex.Lock()
	fc.cache[index] = line
	
	// 如果缓存超过大小，清理最旧的条目
	if len(fc.cache) > fc.cacheSize {
		fc.evictOldEntries()
	}
	fc.cacheMutex.Unlock()
	
	// 触发异步预加载
	select {
	case fc.prefetchQueue <- index:
	default:
		// 如果队列满，跳过
	}
	
	return line
}

// loadLineFromFile 从文件加载指定行
func (fc *FileCache) loadLineFromFile(index int) *TraceLine {
	if index < 0 || index >= len(fc.linePositions) {
		return nil
	}
	
	file, err := os.Open(fc.filename)
	if err != nil {
		return nil
	}
	defer file.Close()
	
	// 定位到行起始位置
	_, err = file.Seek(fc.linePositions[index], 0)
	if err != nil {
		return nil
	}
	
	// 读取该行
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		lineText := scanner.Text()
		traceLine, err := ParseLine(lineText)
		if err != nil {
			fmt.Printf("解析第 %d 行失败: %v\n", index, err)
			return nil
		}
		return traceLine
	}
	
	return nil
}

// prefetchWorker 预加载工作协程
func (fc *FileCache) prefetchWorker() {
	for {
		select {
		case index := <-fc.prefetchQueue:
			fc.prefetchAround(index)
		case <-fc.stopPrefetch:
			return
		}
	}
}

// prefetchAround 预加载指定行周围的行
func (fc *FileCache) prefetchAround(index int) {
	fc.prefetchMutex.Lock()
	defer fc.prefetchMutex.Unlock()
	
	start := index - fc.prefetchWindow/2
	if start < 0 {
		start = 0
	}
	end := start + fc.prefetchWindow
	if end > fc.totalLines {
		end = fc.totalLines
	}
	
	// 批量预加载
	for i := start; i < end; i++ {
		// 检查是否已经在缓存中
		fc.cacheMutex.RLock()
		_, exists := fc.cache[i]
		fc.cacheMutex.RUnlock()
		
		if !exists {
			line := fc.loadLineFromFile(i)
			if line != nil {
				fc.cacheMutex.Lock()
				fc.cache[i] = line
				
				// 如果缓存超过大小，清理最旧的条目
				if len(fc.cache) > fc.cacheSize {
					fc.evictOldEntries()
				}
				fc.cacheMutex.Unlock()
			}
		}
	}
}

// evictOldEntries 清理最旧的缓存条目
func (fc *FileCache) evictOldEntries() {
	// 简单的随机清理策略：清理一半最旧的条目
	targetSize := fc.cacheSize / 2
	
	// 找到最早的访问时间（简化版：随机清理）
	// 在实际应用中，可以维护访问时间戳
	for k := range fc.cache {
		if len(fc.cache) <= targetSize {
			break
		}
		delete(fc.cache, k)
	}
}

// Total 返回总行数
func (fc *FileCache) Total() int {
	return fc.totalLines
}

// Clear 清空缓存
func (fc *FileCache) Clear() {
	fc.cacheMutex.Lock()
	fc.cache = make(map[int]*TraceLine)
	fc.cacheMutex.Unlock()
}

// Close 关闭缓存
func (fc *FileCache) Close() {
	fc.stopPrefetch <- true
	close(fc.prefetchQueue)
	close(fc.stopPrefetch)
}
