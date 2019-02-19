package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func isMultiRegistered(s *school, student, course string) bool {
	for _, v := range s.courses {
		if course != v.c.Name {
			_, ok := v.students[student]
			if ok {
				return true
			}
		}
	}

	return false
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	school := getSchool(r.FormValue("school"))
	student := r.FormValue("student")
	course := r.FormValue("course")
	if student == "" || course == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	errCode := 1
	errMsg := "报名失败"
	school.m.Lock()
	if !school.started {
		errMsg = "报名未开始"
	} else if isMultiRegistered(school, student, course) {
		errMsg = "禁止报多门课"
	} else {
		for _, v := range school.courses {
			if course == v.c.Name  {
				if v.c.Number < v.c.Total {
					if _, ok := v.students[student]; ok {
						errMsg = "重复报名"
					} else {
						v.c.Number += 1
						v.students[student] = true
						school.registerDb(student, v.c)
						errCode = 0
						errMsg = "报名成功"
					}
				} else {
					errMsg = "已报满"
				}
				break
			}
		}
	}
	school.m.Unlock()

	w.Write([]byte(fmt.Sprintf(`{"errCode":%d,"errMsg":"%s"}`, errCode, errMsg)))
}

func handleCancel(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 3 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	school := getSchool(r.FormValue("school"))
	student := r.FormValue("student")
	course := r.FormValue("course")
	if student == "" || course == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	errCode := 1
	errMsg := "取消失败"
	school.m.Lock()
	for _, v := range school.courses {
		if course == v.c.Name  {
			if _, ok := v.students[student]; ok {
				v.c.Number -= 1
				delete(v.students, student)
				errCode = 0
				errMsg = "取消成功"
				school.unRegisterDb(student, course)
			}
			break
		}
	}
	school.m.Unlock()

	w.Write([]byte(fmt.Sprintf(`{"errCode":%d,"errMsg":"%s"}`, errCode, errMsg)))
}

func gradeFilter(grades []int, grade int) bool {
	for _, v := range grades {
		if v == grade {
			return true
		}
	}
	return false
}

func getGrade(year int) int {
	t := time.Now()
	grade := t.Year() - 2000 - year
	if grade >= 0 && grade <= 5 {
		if t.Month() >= 9 {
			grade++
		}
		return grade
	}

	return 0
}

func handleCourse(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	school := getSchool(r.FormValue("school"))
	student := r.FormValue("student")
	if student == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	year, _ := strconv.ParseInt(student[0:2], 10 ,32)
	grade := getGrade(int(year))
	var cl = courseList{[]course{}}
	school.m.RLock()
	for _, v := range school.courses {
		if gradeFilter(v.c.Grade, grade) {
			cl.Data = append(cl.Data, v.c)
		}
	}
	school.m.RUnlock()
	b, err := json.Marshal(&cl)
	if err != nil {
		log.Println(err)
		return
	}
	w.Write(b)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	school := getSchool(r.FormValue("school"))
	if school.started {
		w.Write([]byte(fmt.Sprintf(`{"status":"started","courseTag":"%s"}`, school.courseTag)))
	} else {
		w.Write([]byte(fmt.Sprintf(`{"status":"notStarted","courseTag":"%s"}`, school.courseTag)))
	}
}

func handleRegisterInfo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	school := getSchool(r.FormValue("school"))
	student := r.FormValue("student")
	if student == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	course := ""
	school.m.RLock()
	for _, v := range school.courses {
		_, ok := v.students[student]
		if ok {
			course = v.c.Name
			break
		}
	}
	school.m.RUnlock()

	w.Write([]byte(fmt.Sprintf(`{"course":"%s"}`, course)))
}

func handleRegisterHistory(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	school := getSchool(r.FormValue("school"))
	student := r.FormValue("student")
	if student == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b, _ := school.getRegisterHistory(student)
	w.Write(b)
}

func handleSetTimer(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Write([]byte("Not Implemented"))
}

func handleGetTimer(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	school := getSchool(r.FormValue("school"))
	type CourseTimer struct {
		Name string `json:"name"`
		Time string `json:"time"`
	}
	timers := struct{
		Data []CourseTimer `json:"data"`
	}{[]CourseTimer{}}
	mutexTimers.Lock()
	for k := range tHandlers {
		if c, ok := k.(*CourseStartHandler); ok {
			if c.s == school {
				timers.Data = append(timers.Data,
					CourseTimer{c.name, formatTime(c.seconds)})
			}
		}
	}
	mutexTimers.Unlock()
	b, _ := json.Marshal(&timers)
	w.Write(b)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	school := getSchool(r.FormValue("school"))
	student := r.FormValue("student")
	if student == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	errCode := 0
	name, avatar, err := school.getStudentProfile(student)
	if err != nil {
		errCode = 1
	}
	w.Write([]byte(fmt.Sprintf(`{"errCode":%d,"name":"%s","avatar":"%s"}`, errCode, name, avatar)))
}

func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil || len(r.Form) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	code := r.FormValue("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Write([]byte("Not implemented"))
}
