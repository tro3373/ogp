package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/pkg/errors"
)

const DEBUG = false
const MULTI = 2

func main(args []string) {
	err := handleArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	debug("Done")
}

type Task struct {
	no          int
	ctx         context.Context
	wg          *sync.WaitGroup
	queue       chan string
	resultQueue chan *TaskResult
}

func NewTask(no int, ctx context.Context, wg *sync.WaitGroup, queue chan string, resultQueue chan *TaskResult) *Task {
	wg.Add(1)
	return &Task{
		no:          no,
		ctx:         ctx,
		wg:          wg,
		queue:       queue,
		resultQueue: resultQueue,
	}
}

func (tr *Task) String() string {
	return fmt.Sprintf("Task: { no:%d }", tr.no)
}

type TaskResult struct {
	url string
	og  *opengraph.OpenGraph
	err error
}

func NewTaskResult(url string, og *opengraph.OpenGraph, err error) *TaskResult {
	return &TaskResult{
		url: url,
		og:  og,
		err: err,
	}
}

func (tr *TaskResult) String() string {
	return fmt.Sprintf("TaskResult: {url:%s, og:%v, err:%v}", tr.url, tr.og, tr.err)
}

func handleArgs(args []string) error {
	urls := getUrlsFromStdinOrArgs(args)
	if len(urls) == 0 {
		return fmt.Errorf("No url provided")
	}
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	queue := make(chan string)
	resultQueue := make(chan *TaskResult)

	for i := 0; i < MULTI; i++ {
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

	return handleResult(&wg, urls, resultQueue)
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
	for _, arg := range args {
		urls = append(urls, arg)
	}
	return urls
}

func handleTask(task *Task) {
	debug("==> [Worker] Start", task)
	defer func() {
		debug("==> [Worker] Defer!", task)
		task.wg.Done()
		// debug("==> [worker] closing queue!", task)
		// close(task.queue)
		// debug("==> [worker] closing result queue!", task)
		// close(task.resultQueue)
	}()
	for {
		select {
		case <-task.ctx.Done():
			debug("==> [Worker] Receive ctxDone!", task)
			return
		case url := <-task.queue:
			//  URL取得処理
			debug("==> [Worker] Receive url!", url, task)
			og, err := handleUrl(url)
			// debug("==> [Worker] Sleeping", url, task)
			// time.Sleep(5 * time.Second)
			// debug("==> [Worker] Sleeping Done", url, task)
			task.resultQueue <- NewTaskResult(url, og, err)
			// debug("==> [Worker] Complete url!", url, task)
		}
	}
}

func handleUrl(url string) (*opengraph.OpenGraph, error) {
	var reader io.Reader
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to handle url:%s", url)
	}
	reader = resp.Body
	defer resp.Body.Close()
	og := opengraph.NewOpenGraph()
	if err := og.ProcessHTML(reader); err != nil {
		return nil, errors.Wrapf(err, "Failed to ProcessHTML url:%s", url)
	}
	return og, nil
}

func handleResult(wg *sync.WaitGroup, urls []string, resultQueue chan *TaskResult) error {
	ogs := []*opengraph.OpenGraph{}
	errors := []error{}
	count := 0
	for tr := range resultQueue {
		count++
		debug("=> [Result] Receive tr:", tr)
		if tr.og != nil {
			ogs = append(ogs, tr.og)
		}
		if tr.err != nil {
			debug("=> [Result] Error exist tr:", tr)
			errors = append(errors, tr.err)
		}
		if count == len(urls) {
			debug("=> [Result] Closing resultQueue")
			close(resultQueue)
		}
	}
	if len(errors) > 0 {
		return errors[0]
	}
	err := printResult(ogs)
	if err != nil {
		return err
	}
	debug("wg.Waiting..")
	wg.Wait() //  すべてのgoroutineが終了するのを待つ
	debug("wg.Wait done..")

	return nil
}

func printResult(ogs []*opengraph.OpenGraph) error {
	var target any = ogs
	if len(ogs) == 1 {
		target = ogs[0]
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
	fmt.Fprintln(os.Stderr, a...)
}
