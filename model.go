package main

import (
	"log"
	"sync"
	"time"
)

type course struct {
	Name    string `json:"name"`
	Teacher string `json:"teacher"`
	Total   int    `json:"total"`  //总人数
	Number  int    `json:"number"` //已报人数
	Grade   []int  `json:"grade"`  //适合年级
}

type courseList struct {
	Data []course `json:"data"`
}

type courseObj struct {
	students map[string]bool //已报名的学生
	c        course
}

type registerData struct {
	Student   string `json:"student"`
	Course    string `json:"course"`
	Teacher   string `json:"teacher"`
	TimeStamp int64  `json:"timestamp"`
}

type school struct {
	m         sync.RWMutex
	name      string
	courses   []*courseObj
	started   bool   //报名是否已经开始
	courseTag string //正在报名的课程类别名称，例如：数学课
}

var mutexSchool sync.RWMutex
var schools = map[string]*school{}

func NewCourseObj(name, teacher string, total int, grade []int) *courseObj {
	return &courseObj{students: map[string]bool{}, c: course{name, teacher, total, 0, grade}}
}

func init() {
	go dbRoutine()
}

func getSchool(name string) *school {
	if name == "" {
		return nil
	}

	mutexSchool.RLock()
	s := schools[name]
	mutexSchool.RUnlock()
	if s != nil {
		return s
	}

	//在大多数情况下程序不会执行到这里，只有极端情况下2个以上协程
	//走到这里并且只有一个会抢到写锁并且创建school对象，所以其余的协程
	//被唤醒后需要检查是否school对象是否已经被创建
	mutexSchool.Lock()
	defer mutexSchool.Unlock()
	if s = schools[name]; s != nil {
		return s
	}
	s = &school{name: name, courses: []*courseObj{}, started: false}
	schools[name] = s
	return s
}

func (s *school) loadCourses(name, table string) error {
	courses, err := dbClient.loadCourses(s.name, table)
	if err != nil {
		log.Println(err)
		return err
	}
	s.m.Lock()
	s.courses = courses
	s.started = false
	s.courseTag = name
	s.m.Unlock()
	return nil
}

func (s *school) getRegisterHistory(student string) ([]byte, error) {
	return dbClient.getRegisterHistory(s.name, student)
}

func (s *school) getStudentProfile(student string) (string, string, error) {
	return dbClient.getStudentProfile(s.name, student)
}

type chanHandler interface {
	handle()
}

type chanRegister struct {
	db   string
	data registerData
}

func (self *chanRegister) handle() {
	dbClient.registerCourse(self.db, self.data.Student,
		self.data.Course, self.data.Teacher, self.data.TimeStamp)
}

type chanUnRegister struct {
	db      string
	student string
	course  string
}

func (self *chanUnRegister) handle() {
	dbClient.unRegisterCourse(self.db, self.student, self.course)
}

//channel 的缓冲大小直接影响响应性能，可以根据情况调节缓冲大小
var dbChannel = make(chan chanHandler, 20000)

func dbRoutine() {
	for {
		handler := <-dbChannel
		handler.handle()
	}
}

func (s *school) registerDb(student string, c course) {
	dbChannel <- &chanRegister{
		db:   s.name,
		data: registerData{student, c.Name, c.Teacher, time.Now().Unix()},
	}
}

func (s *school) unRegisterDb(student, course string) {
	dbChannel <- &chanUnRegister{s.name, student, course}
}
