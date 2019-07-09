package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

func ERR_EXIST(id string) string {
	return fmt.Sprintf("The task %v already exists.", id)
}
func ERR_NOTFOUND(id string) string {
	return fmt.Sprintf("The task %v is not found.", id)

}

const (
// ERR_EXIST    = "The task print-time already exists."
// ERR_NOTFOUND = "The task print-time is not found."
)

//Task will be put in Crontab's queue
type Task struct {
	Id     string   `json:"id"`
	Cmd    string   `json:"cmd,omitempty"`
	Args   []string `json:"args,omitempty"`
	ticker *time.Ticker

	Interval int `josn:"interval,omitempty"`
}

func (task *Task) Run() {
	go func() {
		for t := range task.ticker.C {
			cmd := exec.Command(task.Cmd, task.Args...)
			cmd.Stdout = os.Stdout
			cmd.Run()
			fmt.Println(t)
		}

	}()
	// log.Printf("Execute command: %v, Args: %v", task.Cmd, task.Args)

}
func (task *Task) Stop() {
	fmt.Println("called stop ")
	// task.cancel()
	task.ticker.Stop()
}

type Cron struct {
	Tasks map[string]*Task
	mutex sync.Mutex
}

var globalCron Cron

type Response struct {
	Ok    bool   `json:"ok"`
	Id    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

type ByteSize float64

const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
)

func cronHandler(w http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodPost {
		var task Task
		data, _ := ioutil.ReadAll(req.Body)
		err := json.Unmarshal(data, &task)
		taskID := task.Id
		if err != nil {
			log.Fatalf("unmarshal failed:%v", err)
			w.WriteHeader(200)
			w.Write(nil)
			return
		}
		globalCron.mutex.Lock()
		defer globalCron.mutex.Unlock()

		if _, ok := globalCron.Tasks[taskID]; !ok {
			task.ticker = time.NewTicker(time.Duration(task.Interval) * time.Millisecond)
			//task.StopSymbol = make(chan struct{}) //initialize

			globalCron.Tasks[task.Id] = &task

			task.Run()
			response := &Response{
				Ok: true,
				Id: task.Id,
			}
			resData, resErr := json.Marshal(response)
			if resErr != nil {
				log.Fatalf("marshal failed: %v", resErr)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(resData)
		} else {

			response := &Response{
				Ok:    false,
				Error: ERR_EXIST(task.Id),
			}
			resData, resErr := json.Marshal(response)
			if resErr != nil {
				log.Fatalf("marshal failed: %v", resErr)
			}
			w.WriteHeader(http.StatusConflict)
			w.Write(resData)
		}

	}
	if req.Method == http.MethodDelete {
		fmt.Println("check delete")
		var task Task
		data, rErr := ioutil.ReadAll(req.Body)
		if rErr != nil {
			log.Fatalf("ioutil read body failed:%v", rErr)
		}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Fatalf("unmarshal failed:%v", err)
			w.WriteHeader(http.StatusOK)
			w.Write(nil)
			return
		}
		globalCron.mutex.Lock()

		defer globalCron.mutex.Unlock()
		if taskInQ, ok := globalCron.Tasks[task.Id]; !ok {

			response := &Response{
				Ok:    false,
				Error: ERR_NOTFOUND(task.Id),
			}
			resData, resErr := json.Marshal(response)
			if resErr != nil {
				log.Fatalf("marshal failed: %v", resErr)
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write(resData)
		} else {

			taskInQ.Stop()
			// stats := new(runtime.MemStats)
			// previous := stats.Alloc
			// runtime.ReadMemStats(stats)
			// fmt.Printf("Alloc at startup:\t\t%5.4fMB - delta:%5.4fMB\n", ByteSize(stats.Alloc)/MB, ByteSize(stats.Alloc-previous)/MB)

			delete(globalCron.Tasks, taskInQ.Id) //delete
			taskInQ = nil
			//Check memory leak
			// stats = new(runtime.MemStats)
			// runtime.ReadMemStats(stats)
			// fmt.Printf("Alloc After Delete:\t\t%5.4fMB - delta:%5.4fMB\n", ByteSize(stats.Alloc)/MB, ByteSize(stats.Alloc-previous)/MB)

			response := &Response{
				Ok: true,
				Id: task.Id,
			}

			resData, resErr := json.Marshal(response)
			if resErr != nil {
				log.Fatalf("marshal failed: %v", resErr)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(resData)

		}
	}

}

func main() {
	for len(os.Args) != 2 {
		fmt.Println("Please use: ./goCron [port]")
		return
	}
	globalCron.Tasks = make(map[string]*Task)
	http.HandleFunc("/", cronHandler)
	http.ListenAndServe(":"+os.Args[1], nil)

}
