package main

import (
	"bufio"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/valyala/fasthttp"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const BufStr = 10000
const CountWorker = 5000

func main() {
	var wg sync.WaitGroup

	countProc := runtime.NumCPU()
	if countProc == 1 {
		runtime.GOMAXPROCS(2)
	}

	t := time.Now().Format(time.RFC3339)
	ch := make(chan string, BufStr)
	result := make(chan string, 0)

	args := os.Args
	dataBaseFileName := "./RU_Domains_ru-tld.ru"
	if len(args) > 1 {
		dataBaseFileName = args[1]
	}

	defer close(ch)
	defer close(result)
	go readFile(&wg, dataBaseFileName, ch)
	go writeFile("./result_"+t+".log", result)
	s := spinner.New(spinner.CharSets[35], 100*time.Millisecond)
	s.Start()
	for i := 0; i < CountWorker; i++ {
		wg.Add(1)
		go worker(&wg, ch, result)
		wg.Done()
	}
	wg.Wait()
	s.Stop()
}

func worker(wg *sync.WaitGroup, ch chan string, result chan string) {
	for {
		select {
		case s, _ := <-ch:
			sAry := strings.Split(s, "\t")
			//log to console
			fmt.Println(sAry[0])
			if doRequest(sAry[0]) {
				result <- sAry[0]
			}
			wg.Done()
		}
	}
}

func doRequest(url string) bool {
	timeout := time.Millisecond * 3000
	client := &fasthttp.Client{}
	statusTsl, bodyTsl, errTsl := client.GetTimeout(nil, "https://"+url+"/.git/config", timeout)
	status, body, err := client.GetTimeout(nil, "http://"+url+"/.git/config", timeout)
	if status == http.StatusOK || statusTsl == http.StatusOK {
		if len(bodyTsl) > 0 && bodyTsl[0] == 91 {
			return true
		}
		if len(body) > 0 && body[0] == 91 {
			return true
		}
	}
	if err != nil || errTsl != nil {
		return false
	}
	return false
}

func readFile(wg *sync.WaitGroup, file string, ch chan string) {
	f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		wg.Add(1)
		ch <- sc.Text()
	}
	if err := sc.Err(); err != nil {
		return
	}
	return
}

func writeFile(file string, ch chan string) {
	//log to console
	fmt.Println("Create file result" + file)
	f, err := os.Create(file)
	defer f.Close()
	if err != nil {
		return
	}
	for {
		select {
		case s, more := <-ch:
			if more {
				_, err := f.WriteString(s + "\n")
				if err != nil {
					return
				}
			} else {
				return
			}
		}
	}
}
