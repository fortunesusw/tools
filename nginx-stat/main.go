package main

import (
	"strings"
	"strconv"
	"bufio"
	"os"
	"sync"
	"fmt"
	"encoding/json"
	"time"
)

const (
	Z_5 = "[0-5]"
	LT_20 = "(5-20]"
	GT_20 = "(20-]"

	LT_1 = "[0-1]"
	O_5 = "[1-5]"
	GT_5 = "(5-]"
)

type counter struct {
	sync.RWMutex
	C map[string]int `json:"counter"`
}

type FBEnd struct {
	Status                *counter `json:"status"`
	ResponseTimeHistogram *counter `json:"responseTimeHistogram"`
}

type Report struct {
	Domain   string `json:"domain"`
	Total    int `json:"total"`
	FrontEnd *FBEnd `json:"frontEnd"`
	BackEnd  *FBEnd `json:"backEnd"`
}

func parse(line string) {
	defer func() {
		//mutex.Unlock()
		if err := recover(); err != nil {
		}
	}()
	values := strings.Split(line, "  ")

	domain := values[3]
	frontEndStatusCode := values[5]
	backEndStatusCode := values[len(values) - 3]
	frontEndResponseTime := parseFloat(values[7])
	backEndResponseTime := parseFloat(values[len(values) - 2])

	mutex.Lock()
	report, exist := reports[domain]
	if !exist {
		report = &Report {
			Domain: domain,
			Total: 0,
			FrontEnd: &FBEnd{
				Status: &counter{C: make(map[string]int)},
				ResponseTimeHistogram: &counter{C: map[string]int{Z_5:0, LT_20:0, GT_20:0}},
			},
			BackEnd: &FBEnd{
				Status: &counter{C: make(map[string]int)},
				ResponseTimeHistogram: &counter{C: map[string]int{LT_1:0, O_5:0, GT_5:0}},
			},
		}
		reports[domain] = report
	}
	report.Total++
	mutex.Unlock()

	report.FrontEnd.Status.Lock()
	report.FrontEnd.Status.C[frontEndStatusCode]++
	report.FrontEnd.Status.Unlock()

	report.BackEnd.Status.Lock()
	report.BackEnd.Status.C[backEndStatusCode]++
	report.BackEnd.Status.Unlock()

	report.FrontEnd.ResponseTimeHistogram.Lock()
	if frontEndResponseTime >= 0 && frontEndResponseTime <= 5000 {
		report.FrontEnd.ResponseTimeHistogram.C[Z_5]++
	} else if frontEndResponseTime <= 20000 {
		report.FrontEnd.ResponseTimeHistogram.C[LT_20]++
	} else if frontEndResponseTime > 20000 {
		report.FrontEnd.ResponseTimeHistogram.C[GT_20]++
	}
	report.FrontEnd.ResponseTimeHistogram.Unlock()

	report.BackEnd.ResponseTimeHistogram.Lock()
	if backEndResponseTime >= 0 && backEndResponseTime <= 1000 {
		report.BackEnd.ResponseTimeHistogram.C[LT_1]++
	} else if backEndResponseTime <= 5000 {
		report.BackEnd.ResponseTimeHistogram.C[O_5]++
	} else if backEndResponseTime > 5000 {
		report.BackEnd.ResponseTimeHistogram.C[GT_5]++
	}
	report.BackEnd.ResponseTimeHistogram.Unlock()
}

func parseInt(value string, bit int) int32 {
	if len(value) > 0 && value != "-" {
		int_value, err := strconv.ParseInt(value, 10, bit)
		if err != nil {
			return 0;
		} else {
			return int32(int_value)
		}
	}
	return 0
}

func parseFloat(value string) int32 {
	if len(value) > 0 && value != "-" {
		float_value, err := strconv.ParseFloat(value, 10)
		if err != nil {
			return 0;
		} else {
			return int32(float_value * 1000)
		}
	}
	return 0
}

var reports = make(map[string]*Report)
var mutex = &sync.Mutex{}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		go func(line string) {
			parse(line)
		}(line);
	}

	time.Sleep(100 * time.Millisecond)
	mutex.Lock()
	b, err := json.Marshal(reports)
	mutex.Unlock()
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
	}
	fmt.Fprintf(os.Stdout, string(b))
}