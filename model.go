package main

import (
	"sync"
)

type course struct {
	Name string `json:"name"`
	Teacher string `json:"teacher"`
	Total int `json:"total"`  //总人数
	Number int `json:"number"` //已报人数
	Grade []int `json:"grade"` //适合年级
}

type courseList struct {
	Data []course `json:"data"`
}

type courseObj struct {
	students map[string]bool //已报名的学生
	c course
}

type school struct {
	m sync.Mutex
	name string
	courses []*courseObj
	started bool //报名是否已经开始
}

var mutexSchool sync.Mutex
var schools = map[string]*school{}

func NewCourseObj(name, teacher string, total int, grade []int) *courseObj {
	return &courseObj{students: map[string]bool{}, c: course{name, teacher, total, 0, grade}}
}

func getSchool(name string) *school {
	if name == "" {
		return nil
	}

	mutexSchool.Lock()
	s := schools[name]
	mutexSchool.Unlock()

	if s != nil {
		return s
	}

	s = &school{name: name, courses: []*courseObj{}, started:false}
	if err := s.loadCourses(); err != nil {
		return nil
	}

	mutexSchool.Lock()
	schools[name] = s
	mutexSchool.Unlock()
	return s
}
