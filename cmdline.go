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

type CLIContinue func()

func (h CLIContinue) Handle() int {
	h()
	return Continue()
}

type quit struct {}
func (h quit) Handle() int {
	return Quit()
}
func CLIQuit() CLIHandler {
	return quit{}
}

func Quit() int {
	return 0
}

func Continue() int {
	return 1
}

func formatInt(n int64) string {
	if n >= 10 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("0%d", n)
}
func formatTime(seconds int64) string {
	hour := seconds / 3600
	minute := (seconds - hour*3600) / 60
	second := seconds - hour*3600 - minute*60
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
	s             *school
	name          string //课程类别名称，例如：数学课
	table         string //课程所在的数据库表
	seconds       int64  //离报课开始的时间
	secondsToLoad int64  //加载课程距离报名开始的时间
}

func (self *CourseStartHandler) handle() int {
	self.seconds -= 1
	if self.seconds == self.secondsToLoad {
		self.s.loadCourses(self.name, self.table)
	}

	if self.seconds <= 0 {
		ColorGreen(fmt.Sprintf("\n%s报名已开始...", self.name))
		self.s.started = true
		return Quit()
	}

	return Continue()
}

func checkTimer(s *school, seconds int64) (bValid bool) {
	abs := func(a, b int64) int64 {
		if a > b {
			return a - b
		}
		return b - a
	}
	bValid = true
	mutexTimers.Lock()
	for k := range tHandlers {
		if c, ok := k.(*CourseStartHandler); ok {
			//报名开始时间的间隔不能少于30分钟，容忍误差，取1795秒近似30分钟
			if c.s == s && abs(c.seconds, seconds) < 1795 {
				bValid = false
				break
			}
		}
	}
	mutexTimers.Unlock()
	return bValid
}

func SetStartTime(s *school, name, table string) {

	fmt.Print(fmt.Sprintf("输入%s报名开始时间<eg. 18:30>: ", name))
	input := ziphttp.ReadInput()
	match, _ := regexp.MatchString(`^\d+:\d+$`, input)
	if !match {
		ColorRed("设置失败：时间格式错误")
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
		ColorRed("设置失败：不能早于当前时间")
		return
	}
	seconds := (hour-nowH)*3600 + (minute-nowM)*60 - nowS
	if !checkTimer(s, seconds) {
		ColorRed("设置失败：与已有报名的开始时间间隔不能少于30分钟")
		return
	}

	ColorRed(fmt.Sprintf("设置成功：%s报名将在 %s 后开始\n", name, formatTime(seconds)))

	//如果离报名开始的时间小于sToLoad则立刻加载课程
	const sToLoad = 300 //默认5分钟
	h := &CourseStartHandler{s, name, table, seconds, sToLoad}
	if seconds <= sToLoad {
		s.loadCourses(name, table)
	}
	RegisterTHandler(h)
	return
}

func course01() {
	s := getSchool("mbxsj")
	SetStartTime(s, "模块课", "course")
}

func course02() {
	s := getSchool("mbxsj")
	SetStartTime(s, "拓展课", "course02")
}

func test() {
	s := getSchool("mbxsj")
	h := &CourseStartHandler{s, "拓展课", "course02", 1, 0}
	RegisterTHandler(h)
}

var CmdLineHandler = map[string]CLIHandler{
	"1":    CLIContinue(course01),
	"2":    CLIContinue(course02),
	"3":    CLIQuit(),
	"test": CLIContinue(test),
}
