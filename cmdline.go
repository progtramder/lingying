package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"ziphttp"
)

const prompt = `1. 设置报名开始时间
2. 退出`

type CLIHandler interface {
	Handle() int
}

type Handler func() int

func (h Handler) Handle() int {
	return h()
}

func Quit() int {
	return 0
}

func Continue() int {
	return 1
}


func formatInt(n int64) string{
	if n >= 10 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("0%d", n)
}
func formatTime(seconds int64) string {
	hour := seconds / 3600
	minute :=  (seconds - hour * 3600) / 60
	second := seconds - hour * 3600 - minute * 60
	return formatInt(hour) + ":" + formatInt(minute) + ":" + formatInt(second)
}

var timers = map[*school]*int64 {} //key:value - 学校:离开始的时间
var mutexTimers sync.Mutex
func timerHandler() {
	mutexTimers.Lock()
	for s, t := range timers {
		*t -= 1
		fmt.Print(fmt.Sprintf("\r%s ", formatTime(*t)))
		if *t <= 0 {
			delete(timers, s)
			ColorGreen("\n报名已开始...")
			s.started = true
		}
	}
	mutexTimers.Unlock()

	time.AfterFunc(time.Second, timerHandler)
}

func SetStartTime() int {
	s := getSchool("mbxsj")
	if s == nil {
		log.Println("数据库错误")
		return Continue()
	}
	if !s.started {
		mutexTimers.Lock()
		delete(timers, s)
		mutexTimers.Unlock()

		fmt.Print("输入开始时间<eg. 18:30>: ")
		input := ziphttp.ReadInput()
		match, _ := regexp.MatchString(`^\d+:\d+$`, input)
		if !match {
			ColorRed("*时间格式错误*")
			return Continue()
		}

		timeString := strings.Split(input, ":")
		hour, _ := strconv.ParseInt(timeString[0], 10, 32)
		minute, _ := strconv.ParseInt(timeString[1], 10, 32)
		t := time.Now()
		nowH := int64(t.Hour())
		nowM := int64(t.Minute())
		nowS := int64(t.Second())
		if hour < nowH || (hour == nowH && minute <= nowM) {
			ColorRed("*不能早于当前时间*")
			return Continue()
		}
		seconds := (hour - nowH) * 3600 + (minute - nowM) * 60 - nowS
		mutexTimers.Lock()
		timers[s] = &seconds
		mutexTimers.Unlock()
	} else {
		ColorGreen("报名已开始...\n")
	}

	return Continue()
}


var CmdLineHandler = map[string]CLIHandler{
	"1": Handler(SetStartTime),
	"2": Handler(Quit),
}
