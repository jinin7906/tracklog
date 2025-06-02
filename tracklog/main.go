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

type MainThis struct {
	MonitorMgr   *monitor.MonitorMgr
	ProcessorMgr *processor.ProcessMgr
}

func main() {

	MainThis := new(MainThis)

	configPath := flag.String("config", "config/config.yaml", "config file path")
	flag.Parse()

	//read config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load err: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("config load succ - %d\n", len(cfg.Monitors))

	// monitor to processor
	var wg sync.WaitGroup
	logLineChan := make(chan monitor.LogLine, cfg.Global.MaxGoroutines*2)

	MainThis.MonitorMgr = monitor.NewMonitorMgr(&cfg.Global)
	MainThis.MonitorMgr.Start(&cfg.Monitors, logLineChan, &wg)

	MainThis.ProcessorMgr = processor.NewProcessMgr(&cfg.Global, &cfg.Monitors)
	MainThis.ProcessorMgr.Start(logLineChan, &wg)
	// for _, monCfg := range cfg.Monitors {
	// 	wg.Add(1)
	// 	go func(mCfg config.MonitorConfig) {
	// 		defer wg.Done()
	// 		//MainThis.MonitorMgr.GlobalCfg = cfg.Global // 여기 new 호출하는걸로 바꿔야함
	// 		MainThis.MonitorMgr.LogMonitor(mCfg, logLineChan)
	// 	}(monCfg)
	// }

	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	for logLine := range logLineChan {
	// 		processor.ProcessLogLine(logLine, cfg.Global, cfg.Monitors)
	// 	}
	// 	fmt.Println("log chan stop.")
	// }()

	fmt.Println("start Tracklog service")
	select {}

}
