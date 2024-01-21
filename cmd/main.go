package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Bombardier struct {
	timeout       uint
	targetTimeout uint
	timeoutDelta  uint
	url           string
}

func (cfg *Bombardier) runRequester(respCounterChan, eChan, e500Chan chan int, wg *sync.WaitGroup) {
	// TODO: try approach when you set target RPS instead of target timeout
	for {
		wg.Add(1)
		time.Sleep(time.Duration(cfg.timeout) * time.Microsecond)
		cfg.doRequest(respCounterChan, eChan, e500Chan, wg)
	}
}

func (cfg *Bombardier) doRequest(respCounterChan, eChan, e500Chan chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := http.Get(cfg.url)
	respCounterChan <- 1

	if err != nil {
		eChan <- 1
	} else if resp.StatusCode > 499 {
		e500Chan <- 1
	}
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
			if cfg.timeout > cfg.targetTimeout {
				cfg.timeout -= cfg.timeoutDelta
			}
		case errTick := <-errChan:
			errorsCount += errTick
			cfg.timeout += cfg.timeoutDelta * 2
		case err500Tick := <-err500Chan:
			errors500Count += err500Tick
			cfg.timeout += cfg.timeoutDelta * 2
		default:
			time.Sleep(time.Millisecond)
			// fmt.Printf("waiting data....")
		}

		timeDelta := time.Now().Sub(start)
		if timeDelta >= time.Second {
			start = time.Now()

			fmt.Printf("[%s] RPS: %d; errors: %d; 500codes: %d; timeout: %d mcs \n",
				start.String(), requestsCount, errorsCount, errors500Count, cfg.timeout)

			requestsCount = 0
			errorsCount = 0
			errors500Count = 0
		}
	}
}

func main() {
	// TODO: configure from parameters
	requesterNum := 100
	bombardierCfg := Bombardier{
		timeout:       uint(69),
		timeoutDelta:  uint(69),
		targetTimeout: uint(100),
		url:           "http://localhost:8080",
	}

	requestsChan := make(chan int)
	errChan := make(chan int)
	err500Chan := make(chan int)

	wg := sync.WaitGroup{}

	go bombardierCfg.runStatsCalculator(requestsChan, errChan, err500Chan)

	// TODO: immediately exit after process after SIGTERM
	// TODO: fix bug when RPS goes down and stuck on ~200-300 RPS
	for i := 0; i < requesterNum; i++ {
		bombardierCfg.runRequester(requestsChan, errChan, err500Chan, &wg)
	}
}
