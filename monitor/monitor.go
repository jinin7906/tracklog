package monitor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"tracklog/config"

	"github.com/fsnotify/fsnotify"
)

type LogLine struct {
	MonitorName string
	Content     string
	Timestamp   time.Time
}

// log file monitor
func LogMonitor(cfg config.MonitorConfig, lineChan chan<- LogLine) {
	fmt.Printf("[%s] log monitor start: %s/%s\n", cfg.Name, cfg.Path, cfg.FilenamePattern)

	filePath := filepath.Join(cfg.Path, cfg.FilenamePattern)
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_APPEND, 0660)
	if err != nil {
		fmt.Printf("[%s] file open err: %v\n", cfg.Name, err)
		return
	}
	defer file.Close()

	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Printf("[%s] file seek err: %v\n", cfg.Name, err)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("[%s] watcher create err: %v\n", cfg.Name, err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(filePath)
	if err != nil {
		fmt.Printf("[%s] watcher file add err: %v\n", cfg.Name, err)
		return
	}

	reader := bufio.NewReader(file)

	// last update time check - 행 걸린 여부 체크
	lastUpdateTime := time.Now()
	noUpdateCheckInterval := 10 * time.Second // check time

	ticker := time.NewTicker(noUpdateCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return // watcher err
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// file write event
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							// wait write event
							break
						}
						fmt.Printf("[%s] file read err: %v\n", cfg.Name, err)
						break
					}
					// write log line event send
					lineChan <- LogLine{
						MonitorName: cfg.Name,
						Content:     line,
						Timestamp:   time.Now(),
					}
					lastUpdateTime = time.Now()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return // watcher err
			}
			fmt.Printf("[%s] watcher err: %v\n", cfg.Name, err)
		case <-ticker.C:
			// check log event
			if cfg.EventOnNoUpdate && time.Since(lastUpdateTime) > 30*time.Second { // 타임아웃 시간도 옵션으로 빼야겠다
				fmt.Printf("[%s] time out log event\n", cfg.Name)
			}
		}
	}
}
