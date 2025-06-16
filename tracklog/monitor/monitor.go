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
	"tracklog/manager"

	"github.com/fsnotify/fsnotify"
)

type MonitorMgr struct {
	IsRun     bool
	GlobalCfg config.GlobalConfig

	DataMgr *manager.Mgr
}

func NewMonitorMgr(datamgr *manager.Mgr, _glovalCfg *config.GlobalConfig) *MonitorMgr {
	var mng *MonitorMgr = new(MonitorMgr)
	mng.IsRun = false

	if _glovalCfg != nil {
		mng.GlobalCfg = *_glovalCfg
	} else {
		fmt.Println("[MonitorMgr] NewMonitorMgr: _glovalCfg is nil")
		return nil
	}

	if datamgr != nil {
		mng.DataMgr = datamgr
	} else {
		fmt.Println("[MonitorMgr] NewMonitorMgr: datamgr is nil")
		return nil
	}

	return mng
}

func (This *MonitorMgr) Start(monCfgs *[]config.MonitorConfig, lineChan chan<- manager.LogLine, lineChanSch chan<- manager.LogLine, wg *sync.WaitGroup) bool {

	//monCfgsT := make([]config.MonitorConfig, 0)
	monCfgsMap := make(map[time.Time]config.MonitorConfig)

	for _, monCfg := range *monCfgs {
		if monCfg.Realtime { // 실시간
			wg.Add(1)
			go This.LogMonitorRealTime(monCfg, lineChan, wg)
		} else { // 스케줄
			baseTime := time.Now()
			scheduledTimeKey := baseTime.Add(time.Duration(monCfg.FileCheckTime) * time.Second)
			monCfgsMap[scheduledTimeKey] = monCfg

		}
	}

	if len(monCfgsMap) != 0 {
		wg.Add(1)
		go This.LogMonitorSchedule(monCfgsMap, lineChan)
	}
	return true
}

// log file monitor
func (This *MonitorMgr) LogMonitorRealTime(cfg config.MonitorConfig, lineChan chan<- manager.LogLine, wg *sync.WaitGroup) {

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
					lineChan <- manager.LogLine{
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
				err := This.DataMgr.SendLogToTCP(This.GlobalCfg.EventTCPAddress, fmt.Sprintf("[%s] Extracted: %s", cfg.Name, processedContentForTCP))

				if err != nil {
					fmt.Printf("TCP 로그 전송 중 오류 발생: %v\n", err)
				} else {
					fmt.Println("TCP 로그 전송 성공.")
				}
			}
		}
	}
}

func (This *MonitorMgr) LogMonitorSchedule(cfgMap map[time.Time]config.MonitorConfig, lineChanSch chan<- manager.LogLine) {
	fmt.Println("Starting scheduled log monitor. It will check files periodically.")

	//last line
	fileOffsets := make(map[string]int64)

	for {
		currentTime := time.Now()

		tasksToReschedule := make(map[time.Time]config.MonitorConfig)
		tasksToRemove := []time.Time{}

		for schTime, monCfg := range cfgMap {
			if currentTime.After(schTime) || currentTime.Equal(schTime) {
				fmt.Printf("--- Scheduled task ready for: %s (Scheduled at: %s) ---\n",
					monCfg.Name, schTime.Format("2006-01-02 15:04:05"))

				filePath := filepath.Join(monCfg.Path, monCfg.FilenamePattern)

				currentOffset, exists := fileOffsets[filePath]
				if !exists {
					currentOffset = 0
				}

				file, err := os.OpenFile(filePath, os.O_RDONLY, 0660)
				if err != nil {
					fmt.Printf("[%s] Scheduled file open err: %v\n", monCfg.Name, err)
					// 파일오픈 실패시 고려할게 필요함
					continue
				}

				_, err = file.Seek(currentOffset, io.SeekStart)
				if err != nil {
					fmt.Printf("[%s] Scheduled file seek error to offset %d: %v\n", monCfg.Name, currentOffset, err)
					file.Close()
					continue
				}

				reader := bufio.NewReader(file)
				newOffset := currentOffset

				// 파일사이즈 변동시 처리 로직
				fileStat, statErr := file.Stat()
				if statErr != nil {
					fmt.Printf("[%s] Error getting file stat for %s: %v\n", monCfg.Name, filePath, statErr)
					file.Close()
					continue
				}

				// 파일 변동시 처리 로직
				if newOffset > fileStat.Size() {
					fmt.Printf("  [%s] File %s truncated or recreated. Resetting offset from %d to 0.\n", monCfg.Name, filePath, newOffset)
					newOffset = 0                               // 오프셋 리셋
					_, err = file.Seek(newOffset, io.SeekStart) // 다시 처음부터 시크
					if err != nil {
						fmt.Printf("[%s] File seek error after reset: %v\n", monCfg.Name, err)
						file.Close()
						continue
					}
					reader = bufio.NewReader(file)
				}

				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							break
						}
						fmt.Printf("[%s] Scheduled file read error: %v\n", monCfg.Name, err)
						break
					}

					lineChanSch <- manager.LogLine{
						MonitorName: monCfg.Name,
						Content:     line,
						Timestamp:   time.Now(),
					}
					newOffset += int64(len(line))
				}
				file.Close()

				//update offset
				fileOffsets[filePath] = newOffset

				//update sch time
				nextScheduledTime := time.Now().Add(time.Duration(monCfg.FileCheckTime) * time.Second)
				tasksToReschedule[nextScheduledTime] = monCfg
				tasksToRemove = append(tasksToRemove, schTime) // 처리된 파일 update

				fmt.Printf("  [%s] Task completed. Next scheduled at: %s (New offset: %d)\n",
					monCfg.Name, nextScheduledTime.Format("2006-01-02 15:04:05"), newOffset)
			}
		}

		for _, oldSchTime := range tasksToRemove {
			delete(cfgMap, oldSchTime)
		}
		for newSchTime, newMonCfg := range tasksToReschedule {
			cfgMap[newSchTime] = newMonCfg
		}

		time.Sleep(500 * time.Millisecond)
	}
}
