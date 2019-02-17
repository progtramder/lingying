package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"ziphttp"
)

const prompt = `1. 设置模块课报名开始时间
2. 设置拓展课报名开始时间
3. 退出`

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

var tHandlers = map[THandler]interface{}{} //把map当成list用
var mutexTimers sync.Mutex
func IntervalHandler() {
	deletingHandlers := make([]THandler, 0)
	mutexTimers.Lock()
	for h := range tHandlers {
		if h.handle() == Quit() {
			deletingHandlers = append(deletingHandlers, h)
		}
	}

	for _, h := range deletingHandlers {
		delete(tHandlers, h)
	}
	mutexTimers.Unlock()

	time.AfterFunc(time.Second, IntervalHandler)
}

func RegisterTHandler(handler THandler) {
	mutexTimers.Lock()
	tHandlers[handler] = nil
	mutexTimers.Unlock()
}

type THandler interface {
	handle() int // return value 0: delete the THandler
}

type CourseStartHandler struct {
	 s *school
	 name string //课程类别名称
	 table string //课程所在的数据库表
	 seconds int64 //离报课开始的时间
}

func (self *CourseStartHandler) handle() int {
	self.seconds -= 1
	//剩下5分钟时重新加载即将开始的课程
	if self.seconds == 10 {
		self.s.loadCourses(self.table)
	}

	if self.seconds <= 0 {
		ColorGreen(fmt.Sprintf("\n%s报名已开始...", self.name))
		self.s.started = true
		return Quit()
	}

	return Continue()
}

func SetStartTime(s* school, name, table string) {

	fmt.Print(fmt.Sprintf("输入%s报名开始时间<eg. 18:30>: ", name))
	input := ziphttp.ReadInput()
	match, _ := regexp.MatchString(`^\d+:\d+$`, input)
	if !match {
		ColorRed("*时间格式错误*")
		return
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
		return
	}
	seconds := (hour - nowH) * 3600 + (minute - nowM) * 60 - nowS
	ColorRed(fmt.Sprintf("%s报名将在 %s 后开始\n", name, formatTime(seconds)))
	RegisterTHandler(&CourseStartHandler{s, name, table, seconds})
	return
}

func course01() int {
	s := getSchool("mbxsj")
	SetStartTime(s, "模块课", "course")
	return Continue()
}

func course02() int {
	s := getSchool("mbxsj")
	SetStartTime(s, "拓展课", "course02")
	return Continue()
}

var CmdLineHandler = map[string]CLIHandler{
	"1": Handler(course01),
	"2": Handler(course02),
	"3": Handler(Quit),
}
