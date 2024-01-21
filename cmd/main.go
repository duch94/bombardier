package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func runRequester(respCounterChan, eChan, e500Chan chan int, wg *sync.WaitGroup, timeout *uint) {
	for {
		wg.Add(1)
		time.Sleep(time.Duration(*timeout) * time.Microsecond)
		doRequest(respCounterChan, eChan, e500Chan, wg)
	}
}

func doRequest(respCounterChan, eChan, e500Chan chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := http.Get("http://localhost:8080")
	respCounterChan <- 1

	if err != nil {
		eChan <- 1
	} else if resp.StatusCode > 499 {
		e500Chan <- 1
	}
}

func runStatsCalculator(requestsChan, errChan, err500Chan chan int, timeout *uint) {
	start := time.Now()
	requestsCount := 0
	errorsCount := 0
	errors500Count := 0

	targetTimeout := uint(100)
	timeoutDelta := uint(69)

	for {
		select {
		case requestTick := <-requestsChan:
			requestsCount += requestTick
			if *timeout > targetTimeout {
				*timeout -= timeoutDelta
			}
		case errTick := <-errChan:
			errorsCount += errTick
			*timeout += timeoutDelta * 2
		case err500Tick := <-err500Chan:
			errors500Count += err500Tick
			*timeout += timeoutDelta * 2
		default:
			time.Sleep(time.Millisecond)
			// fmt.Printf("waiting data....")
		}

		timeDelta := time.Now().Sub(start)
		if timeDelta >= time.Second {
			start = time.Now()

			fmt.Printf("[%s] RPS: %d; errors: %d; 500codes: %d; timeout: %d mcs \n",
				start.String(), requestsCount, errorsCount, errors500Count, *timeout)

			requestsCount = 0
			errorsCount = 0
			errors500Count = 0
		}
	}
}

func main() {
	requesterNum := 100

	requestsChan := make(chan int)
	errChan := make(chan int)
	err500Chan := make(chan int)

	wg := sync.WaitGroup{}
	timeout := uint(69) // microseconds

	go runStatsCalculator(requestsChan, errChan, err500Chan, &timeout)

	for i := 0; i < requesterNum; i++ {
		runRequester(requestsChan, errChan, err500Chan, &wg, &timeout)
	}
}
