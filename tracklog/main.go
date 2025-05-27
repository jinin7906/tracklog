package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"tracklog/config"
	"tracklog/monitor"
	"tracklog/processor"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "설정 파일 경로")
	flag.Parse()

	//read config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load err: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("config load succ - %d\n", len(cfg.Monitors))

	// monitor to processor
	logLineChan := make(chan monitor.LogLine, cfg.Global.MaxGoroutines*2)

	var wg sync.WaitGroup

	for _, monCfg := range cfg.Monitors {
		wg.Add(1)
		go func(mCfg config.MonitorConfig) {
			defer wg.Done()
			monitor.LogMonitor(mCfg, logLineChan)
		}(monCfg)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for logLine := range logLineChan {
			processor.ProcessLogLine(logLine, cfg.Global, cfg.Monitors)
		}
		fmt.Println("log chan stop.")
	}()

	fmt.Println("start Tracklog service")

	select {}

}
