package monitor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tracklog/config"
	"tracklog/tcp_client"

	"github.com/fsnotify/fsnotify"
)

type MonitorMgr struct {
	IsRun     bool
	GlobalCfg config.GlobalConfig
}

type LogLine struct {
	MonitorName string
	Content     string
	Timestamp   time.Time
}

func NewMonitorMgr(_glovalCfg *config.GlobalConfig) *MonitorMgr {
	var mng *MonitorMgr = new(MonitorMgr)
	mng.IsRun = false

	if _glovalCfg != nil {
		mng.GlobalCfg = *_glovalCfg
	} else {
		fmt.Println("[MonitorMgr] NewMonitorMgr: _glovalCfg is nil. Using default/empty GlobalConfig.")
	}
	return mng
}

func (This *MonitorMgr) Start(monCfgs *[]config.MonitorConfig, lineChan chan<- LogLine, wg *sync.WaitGroup) bool {

	//var wg sync.WaitGroup

	for _, monCfg := range *monCfgs {
		wg.Add(1)
		go This.LogMonitor(monCfg, lineChan, wg)
	}

	return true
}

// log file monitor
func (This *MonitorMgr) LogMonitor(cfg config.MonitorConfig, lineChan chan<- LogLine, wg *sync.WaitGroup) {

	defer wg.Done()

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
	noUpdateCheckInterval := 10 * time.Second // 10sec 고정

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
			if cfg.LogWriteSec > 0 && time.Since(lastUpdateTime) > time.Duration(cfg.LogWriteSec)*time.Second {
				processedContentForTCP := strings.TrimRight("추가로그 작성 없음", "\n\r")
				err := tcp_client.SendLogToTCP(This.GlobalCfg.EventTCPAddress, fmt.Sprintf("[%s] Extracted: %s", cfg.Name, processedContentForTCP))

				if err != nil {
					fmt.Printf("TCP 로그 전송 중 오류 발생: %v\n", err)
				} else {
					fmt.Println("TCP 로그 전송 성공.")
				}
			}
		}
	}
}
