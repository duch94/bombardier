package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Bombardier struct {
	targetRPS  uint
	timeout    float64
	url        string
	httpClient *http.Client
}

func (cfg *Bombardier) runStatsCalculator(requestsChan, errChan, err500Chan chan int) {
	start := time.Now()
	requestsCount := 0
	errorsCount := 0
	errors500Count := 0

	for {
		select {
		case requestTick := <-requestsChan:
			requestsCount += requestTick
		case errTick := <-errChan:
			errorsCount += errTick
		case err500Tick := <-err500Chan:
			errors500Count += err500Tick
		default:
			time.Sleep(time.Millisecond)
			// fmt.Printf("waiting data....")
		}

		timeDelta := time.Now().Sub(start)
		if timeDelta >= time.Second {
			start = time.Now()

			fmt.Printf("[%s] RPS: %d; errors: %d; 500codes: %d; timeout: %f ms \n",
				start.String(), requestsCount, errorsCount, errors500Count, cfg.timeout)

			requestsCount = 0
			errorsCount = 0
			errors500Count = 0
		}
	}
}

func (cfg *Bombardier) runRequester(respCounterChan, eChan, e500Chan chan int, wg *sync.WaitGroup) {
	cfg.timeout = float64(1000) / float64(cfg.targetRPS)
	defer wg.Done()
	for {
		time.Sleep(time.Duration(cfg.timeout) * time.Millisecond)
		cfg.doRequest(respCounterChan, eChan, e500Chan)
	}
}

func (cfg *Bombardier) doRequest(respCounterChan, eChan, e500Chan chan int) {
	resp, err := http.Get(cfg.url)
	defer resp.Body.Close()

	respCounterChan <- 1
	if err != nil {
		eChan <- 1
	} else if resp.StatusCode > 499 {
		e500Chan <- 1
	}

	err = resp.Body.Close()
	if err != nil {
		eChan <- 1
	}
}

func main() {
	requesterNum := 10
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: requesterNum,
		}}

	bombardier := Bombardier{
		targetRPS:  uint(1000),
		url:        "http://localhost:8080",
		httpClient: client,
	}

	requestsChan := make(chan int)
	errChan := make(chan int)
	err500Chan := make(chan int)

	wg := sync.WaitGroup{}

	go bombardier.runStatsCalculator(requestsChan, errChan, err500Chan)

	// TODO: fix bug when RPS goes down and stuck on ~20-30 RPS
	for i := 0; i < requesterNum; i++ {
		wg.Add(1)
		go bombardier.runRequester(requestsChan, errChan, err500Chan, &wg)
	}
	wg.Wait()
}
