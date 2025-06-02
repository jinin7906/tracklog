package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"tracklog/config"
	"tracklog/monitor"
	"tracklog/tcp_client"
)

type ProcessMgr struct {
	IsRun     bool
	GlobalCfg config.GlobalConfig
	MonCfgs   []config.MonitorConfig
}

// type LogLine struct {
// 	MonitorName string
// 	Content     string
// 	Timestamp   time.Time
// }

func NewProcessMgr(_glovalCfg *config.GlobalConfig, monCfgs *[]config.MonitorConfig) *ProcessMgr {
	var mng *ProcessMgr = new(ProcessMgr)
	mng.IsRun = false

	if _glovalCfg != nil {
		mng.GlobalCfg = *_glovalCfg
	} else {
		fmt.Println("[ProcessMgr] NewMonitorMgr: _glovalCfg is nil. Using default/empty GlobalConfig.")
	}

	if monCfgs != nil {
		mng.MonCfgs = *monCfgs
	} else {
		fmt.Println("[ProcessMgr] NewMonitorMgr: _glovalCfg is nil. Using default/empty GlobalConfig.")
	}

	return mng
}

func (This *ProcessMgr) Start(lineChan chan monitor.LogLine, wg *sync.WaitGroup) bool {

	//var wg sync.WaitGroup
	This.IsRun = true

	wg.Add(1)
	go func() {
		defer wg.Done()
		for logLine := range lineChan {
			This.ProcessLogLine(logLine)
		}
		fmt.Println("log chan stop.")
	}()
	return true
}

// log line recv process
func (This *ProcessMgr) ProcessLogLine(line monitor.LogLine) {

	if This.IsRun == false {
		return
	}

	// find monitor config
	var currentMonitorCfg *config.MonitorConfig
	for _, cfg := range This.MonCfgs {
		if cfg.Name == line.MonitorName {
			currentMonitorCfg = &cfg
			break
		}
	}

	if currentMonitorCfg == nil {
		fmt.Printf("not find monitor config: %s\n", line.MonitorName)
		return
	}

	var extractedContent string
	isMatched := false

	//regex
	if currentMonitorCfg.ExtractType == "regex" {
		re, err := regexp.Compile(currentMonitorCfg.ExtractPattern)
		if err != nil {
			fmt.Printf("[%s] regex err: %v\n", currentMonitorCfg.Name, err)
			return
		}
		if re.MatchString(line.Content) {
			isMatched = true
			//치환할 문자도 옵션처리 할까?
			//matches := re.FindStringSubmatch(line.Content)
			// if len(matches) > 1 {
			// 	extractedContent = strings.Join(matches[1:], " | ")
			// } else {
			// 	extractedContent = matches[0]
			// }
			extractedContent = line.Content
		}
	} else if currentMonitorCfg.ExtractType == "plain" {
		if strings.Contains(line.Content, currentMonitorCfg.ExtractPattern) {
			isMatched = true
			extractedContent = line.Content
		}
	}

	if isMatched {
		fmt.Printf("[%s] extract log: %s\n", currentMonitorCfg.Name, extractedContent)

		// extract save log
		if currentMonitorCfg.SaveExtracted {
			savePath := currentMonitorCfg.SavePath
			if savePath == "" {
				savePath = This.GlobalCfg.DefaultSavePath + "/" + currentMonitorCfg.Name + "_extracted.log"
			}

			processedContent := strings.TrimRight(extractedContent, "\n\r")

			// log time content check
			finalLogLine := ""

			if regexp.MustCompile(This.GlobalCfg.LogTimeMsgRegex).MatchString(processedContent) {
				finalLogLine = processedContent
			} else {
				finalLogLine = fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), processedContent)
			}

			err := appendToFile(savePath, finalLogLine+"\n")
			if err != nil {
				fmt.Printf("[%s] file save error: %v\n", currentMonitorCfg.Name, err)
			} else {
				//succ
			}
		}

		// tcp send
		if This.GlobalCfg.EventTCPEnabled && currentMonitorCfg.EventTCPEnabled {
			processedContentForTCP := strings.TrimRight(extractedContent, "\n\r")
			tcp_client.SendLogToTCP(This.GlobalCfg.EventTCPAddress, fmt.Sprintf("[%s] Extracted: %s", currentMonitorCfg.Name, processedContentForTCP))
		}
	} else {
		// 매칭실패
	}
}

// write tracklog
func appendToFile(filePath, content string) error {
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("dir create err %s: %w", dir, err)
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("tracklog file open err: %s: %w", filePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("tracklog file write err: %s: %w", filePath, err)
	}
	return nil
}
