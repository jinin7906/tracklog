package processor

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"tracklog/config"
	"tracklog/monitor"
)

// log line recv process
func ProcessLogLine(line monitor.LogLine, globalCfg config.GlobalConfig, monitorCfgs []config.MonitorConfig) {
	// find monitor config
	var currentMonitorCfg *config.MonitorConfig
	for _, cfg := range monitorCfgs {
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
			matches := re.FindStringSubmatch(line.Content)
			if len(matches) > 1 {
				extractedContent = strings.Join(matches[1:], " | ")
			} else {
				extractedContent = matches[0]
			}
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
				savePath = globalCfg.DefaultSavePath + "/" + currentMonitorCfg.Name + "_extracted.log"
			}
			err := appendToFile(savePath, fmt.Sprintf("[%s] %s\n", line.Timestamp.Format(time.RFC3339), extractedContent))
			if err != nil {
				fmt.Printf("[%s] file save error: %v\n", currentMonitorCfg.Name, err)
			} else {
				// fmt.Printf("[%s] log save: %s\n", currentMonitorCfg.Name, savePath)
			}
		}

		if globalCfg.EventTCPEnabled && currentMonitorCfg.EventTCPEnabled {
			sendViaTCP(globalCfg.EventTCPAddress, fmt.Sprintf("[%s] Extracted: %s", currentMonitorCfg.Name, extractedContent))
		}
	}
}

// write tracklog
func appendToFile(filePath, content string) error {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("tracklog file open err: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("tracklog file write err: %w", err)
	}
	return nil
}

// tcp send
func sendViaTCP(address, data string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("tcp connect err %s: %v\n", address, err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(data + "\n"))
	if err != nil {
		fmt.Printf("tcp send err %s: %v\n", address, err)
	} else {
		// tcp send ok
	}
}
