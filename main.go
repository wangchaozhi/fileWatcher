package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

func main() {
	// 创建 watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// 防抖缓存
	var mu sync.Mutex
	eventCache := make(map[string]time.Time)
	// 处理状态跟踪
	processing := sync.Map{}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
					continue
				}

				// 检查是否正在处理
				if _, exists := processing.LoadOrStore(event.Name, true); exists {
					fmt.Printf("忽略事件：文件 %s 正在处理\n", event.Name)
					continue
				}

				mu.Lock()
				lastTime, found := eventCache[event.Name]
				now := time.Now()
				if !found || now.Sub(lastTime) > 3*time.Second {
					eventCache[event.Name] = now
					go handleEvent(event.Name, &processing)
				} else {
					// 防抖期间释放处理状态
					processing.Delete(event.Name)
				}
				mu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("监听错误:", err)
			}
		}
	}()

	// 设置监听路径
	targetPath := "A" // ← 替换为你的路径
	err = watcher.Add(targetPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("正在监听文件夹：", targetPath)

	<-make(chan struct{}) // 阻塞主线程
}

// 处理事件
func handleEvent(path string, processing *sync.Map) {
	defer processing.Delete(path) // 处理完成后清除状态
	fmt.Println("检测到变更，等待文件稳定：", path)

	if isFileStable(path, 1*time.Second, 10) {
		fmt.Println("✅ 文件稳定，可以处理：", path)
		// 这里可以执行你自己的逻辑，比如读取文件、移动、上传等
	} else {
		fmt.Println("⚠️ 文件未稳定，尝试延迟检查：", path)
		// 延迟 5 秒后再次检查
		time.Sleep(5 * time.Second)
		if isFileStable(path, 1*time.Second, 3) {
			fmt.Println("✅ 延迟检查确认文件稳定，可以处理：", path)
		} else {
			fmt.Println("⚠️ 延迟检查后文件仍未稳定，跳过处理：", path)
		}
	}
}

// 检查文件在若干次内大小和修改时间是否保持不变
func isFileStable(path string, interval time.Duration, checks int) bool {
	var lastSize int64 = -1
	var lastModTime time.Time

	for i := 0; i < checks; i++ {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("检查失败 (第 %d/%d 次): %v\n", i+1, checks, err)
			return false
		}
		currentSize := info.Size()
		currentModTime := info.ModTime()

		fmt.Printf("检查 %d/%d: 文件大小=%d, 修改时间=%v\n", i+1, checks, currentSize, currentModTime)

		if i == 0 {
			lastSize = currentSize
			lastModTime = currentModTime
		} else if currentSize != lastSize || !currentModTime.Equal(lastModTime) {
			fmt.Printf("文件不稳定: 大小变化 (%d -> %d) 或修改时间变化 (%v -> %v)\n",
				lastSize, currentSize, lastModTime, currentModTime)
			return false
		}
		time.Sleep(interval)
	}

	fmt.Println("文件稳定")
	return true
}
