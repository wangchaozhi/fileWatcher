package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type WatchItem struct {
	File    string `json:"file"`
	Command string `json:"command"`
}

func main() {
	// 读取配置
	items, err := loadWatchItems("fileWatcher.json")
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	// 创建 watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// 映射文件到命令
	commandMap := make(map[string]string)
	for _, item := range items {
		absPath, err := filepath.Abs(item.File)
		if err != nil {
			log.Fatalf("无法解析文件路径: %v", err)
		}
		commandMap[absPath] = item.Command

		// 监听文件所在目录
		dir := filepath.Dir(absPath)
		err = watcher.Add(dir)
		if err != nil {
			log.Fatalf("监听目录失败 %s: %v", dir, err)
		}
		fmt.Printf("监听中: %s\n", dir)
	}

	var mu sync.Mutex
	eventCache := make(map[string]time.Time)
	processing := sync.Map{}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				absPath, _ := filepath.Abs(event.Name)
				if _, watch := commandMap[absPath]; !watch {
					continue
				}

				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
					continue
				}

				if _, exists := processing.LoadOrStore(absPath, true); exists {
					continue
				}

				mu.Lock()
				lastTime, found := eventCache[absPath]
				now := time.Now()
				if !found || now.Sub(lastTime) > 3*time.Second {
					eventCache[absPath] = now
					go handleEvent(absPath, commandMap[absPath], &processing)
				} else {
					processing.Delete(absPath)
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

	<-make(chan struct{}) // 阻塞主线程
}

func loadWatchItems(path string) ([]WatchItem, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []WatchItem
	err = json.Unmarshal(data, &items)
	return items, err
}

func handleEvent(path, command string, processing *sync.Map) {
	defer processing.Delete(path)
	fmt.Println("检测到变更，等待文件稳定：", path)

	if isFileStable(path, 1*time.Second, 10) {
		fmt.Println("✅ 文件稳定，执行命令：", command)
		runCommand(command)
	} else {
		fmt.Println("⚠️ 文件未稳定，延迟重试：", path)
		if isFileStable(path, 1*time.Second, 3) {
			fmt.Println("✅ 延迟确认稳定，执行命令：", command)
			runCommand(command)
		} else {
			fmt.Println("❌ 文件仍不稳定，跳过：", path)
		}
	}
}

func isFileStable(path string, interval time.Duration, checks int) bool {
	var lastSize int64 = -1
	var lastModTime time.Time

	for i := 0; i < checks; i++ {
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		currentSize := info.Size()
		currentModTime := info.ModTime()

		if i == 0 {
			lastSize = currentSize
			lastModTime = currentModTime
		} else if currentSize != lastSize || !currentModTime.Equal(lastModTime) {
			return false
		}
		time.Sleep(interval)
	}
	return true
}

func runCommand(command string) {
	cmd := exec.Command("sh", "-c", command) // 用于兼容 Linux/Mac，Windows 请用 "cmd", "/C", command
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("命令执行失败：%v\n", err)
	}
}
