package main

import "time"

type DaysData struct {
	Days map[string]int `json:"days"`
}

type HCell struct {
	Date string
	N, L int
	Pad  bool
}

type HWeek struct {
	Cells []HCell
	Month string
}

type HeatView struct {
	Weeks []HWeek
}

func level(n int) int {
	switch {
	case n <= 0:
		return 0
	case n <= 2:
		return 1
	case n <= 5:
		return 2
	case n <= 10:
		return 3
	default:
		return 4
	}
}

func buildHeat(d DaysData) HeatView {
	var last time.Time
	for k := range d.Days {
		if t, err := time.Parse("2006-01-02", k); err == nil && t.After(last) {
			last = t
		}
	}
	if last.IsZero() {
		return HeatView{}
	}

	weekEnd := last.AddDate(0, 0, int(time.Saturday-last.Weekday()))
	start := weekEnd.AddDate(0, 0, -(53*7 - 1))
	var v HeatView
	prevMonth := time.Month(0)
	for w := 0; w < 53; w++ {
		wk := HWeek{}
		for dow := 0; dow < 7; dow++ {
			day := start.AddDate(0, 0, w*7+dow)
			key := day.Format("2006-01-02")
			c := HCell{Date: day.Format("2 Jan 2006")}
			if day.After(last) {
				c.Pad = true
			} else {
				c.N = d.Days[key]
				c.L = level(c.N)
			}
			wk.Cells = append(wk.Cells, c)
		}
		if m := start.AddDate(0, 0, w*7).Month(); m != prevMonth {
			wk.Month = m.String()[:3]
			prevMonth = m
		}
		v.Weeks = append(v.Weeks, wk)
	}
	return v
}
