package flood

import (
	"fmt"
	"time"
)

type Result struct {
	success bool
	dt      time.Duration
}

type ResultAggregator struct {
	NSuccesful, NAttempted, LastCount                   int
	ReportName, PastTenseVerb, SingularNoun, PluralNoun string
	Results                                             []Result
	StartTime, LastTime                                 time.Time
	UpdateThreshold                                     int
}

func (r *ResultAggregator) AppendResult(result Result) {
	if result.success {
		r.NSuccesful += 1
	}
	r.NAttempted += 1
	if r.NAttempted%r.UpdateThreshold == 0 {
		delta := r.NSuccesful - r.LastCount
		update := fmt.Sprintf("%s %d/%d in %s (∆=%d/%d %.2f succesful %s per second)", r.PastTenseVerb, r.NSuccesful, r.NAttempted, time.Since(r.StartTime), r.NSuccesful-r.LastCount, r.UpdateThreshold, float64(r.NSuccesful-r.LastCount)/time.Since(r.LastTime).Seconds(), r.PluralNoun)
		if delta == r.UpdateThreshold {
			fmt.Println(greenColor + update + defaultStyle)
		} else {
			fmt.Println(redColor + update + defaultStyle)
		}
		r.LastCount = r.NSuccesful
		r.LastTime = time.Now()
	}
	r.Results = append(r.Results, result)
}

func (r ResultAggregator) PrintReport() {
	if r.NAttempted == 0 {
		return
	}

	dt := time.Since(r.StartTime)
	slowest := time.Duration(0)
	fastest := time.Hour
	totalTime := time.Duration(0)
	for _, result := range r.Results {
		if result.dt > slowest {
			slowest = result.dt
		}
		if result.dt < fastest {
			fastest = result.dt
		}
		totalTime += result.dt
	}

	numFailed := r.NAttempted - r.NSuccesful
	fractionFailed := float64(numFailed) / float64(r.NAttempted) * 100.0

	CyanBanner(r.ReportName)
	fmt.Println("")
	fmt.Printf(cyanColor+"  %s %d/%d (missed %d = %.2f%%) in %s\n"+defaultStyle, r.PastTenseVerb, r.NSuccesful, r.NAttempted, numFailed, fractionFailed, dt)
	fmt.Printf(cyanColor+"  That's %.2f succesful %s per second\n"+defaultStyle, float64(r.NSuccesful)/dt.Seconds(), r.PluralNoun)
	fmt.Printf(cyanColor+"  And    %.2f attempts per second\n"+defaultStyle, float64(r.NAttempted)/dt.Seconds())
	fmt.Printf(cyanColor+"  Fastest %s took %s\n"+defaultStyle, r.SingularNoun, fastest)
	fmt.Printf(cyanColor+"  Slowest %s took %s\n"+defaultStyle, r.SingularNoun, slowest)
	fmt.Printf(cyanColor+"  Mean %s took %s\n"+defaultStyle, r.SingularNoun, totalTime/time.Duration(r.NAttempted))
	fmt.Println("")
}
