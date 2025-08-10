package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

const DEBUG = false
const MULTI = 2

var xClient *XClient

func handle(args []string) {
	// initConfigInner
	level, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err == nil {
		log.SetLevel(level)
	}
	log.Debug("Debug start")

	xClient, err = NewXClient()
	if err != nil {
		log.Warnf("Failed to initialize XClient: %v", err)
	}

	err = handleArgs(args)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	debug("Done")
}

func handleArgs(args []string) error {
	urls := getUrlsFromStdinOrArgs(args)
	if len(urls) == 0 {
		return fmt.Errorf("no url provided")
	}
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	queue := make(chan string)
	resultQueue := make(chan *TaskResult)

	for i := range MULTI {
		task := NewTask(i, ctx, &wg, queue, resultQueue)
		go handleTask(task)
	}
	go func() {
		for _, url := range urls {
			debug("=> [Input] Queing url:", url)
			queue <- url
		}
		debug("=> [Input] Cancel!")
		cancel() // ctxを終了させる
	}()

	taskResults := collectResults(urls, resultQueue)

	debug("wg.Waiting..")
	wg.Wait() //  すべてのgoroutineが終了するのを待つ
	debug("wg.Wait done..")

	return printResult(taskResults)
}

func getUrlsFromStdinOrArgs(args []string) []string {
	var urls []string

	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// pipe
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}
	}
	urls = append(urls, args...)
	return urls
}

func handleTask(task *Task) {
	debug("==> [Worker] Start", task)
	defer func() {
		debug("==> [Worker] Defer!", task)
		task.Wg.Done()
	}()
	for {
		select {
		case <-task.Ctx.Done():
			debug("==> [Worker] Receive ctxDone!", task)
			return
		case url := <-task.Queue:
			//  URL取得処理
			debug("==> [Worker] Receive url!", url, task)
			tr := handleURL(url)
			task.ResultQueue <- tr
		}
	}
}

func collectResults(urls []string, resultQueue chan *TaskResult) []*TaskResult {
	taskResults := []*TaskResult{}
	count := 0
	for tr := range resultQueue {
		count++
		debug("=> [Result] Receive tr:", tr)
		taskResults = append(taskResults, tr)
		if tr.Err != nil {
			debug("=> [Result] Error exist tr:", tr)
		}
		if count == len(urls) {
			debug("=> [Result] Closing resultQueue")
			close(resultQueue)
		}
	}
	return taskResults
}

func printResult(taskResults []*TaskResult) error {
	// エラーのないTaskResultを抽出
	results := make([]*TaskResult, 0, len(taskResults))
	for _, result := range taskResults {
		if result.Err != nil {
			continue
		}
		results = append(results, result)
	}

	// 1件しかない場合は配列ではなく、Objectで出力する
	var target any = results
	if len(results) == 1 {
		target = results[0]
	}

	output, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func debug(a ...any) {
	if !DEBUG {
		return
	}
	log.Debugf("%+v", a...)
}
