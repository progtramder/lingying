package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"log"
	"strconv"
	"strings"
	"time"
)

type database interface {
	init(string) error
	loadCourses(*school) error
	registerCourse(string, string, string, string, int64) error
	unRegisterCourse(string, string, string) error
	getRegisterHistory(string, string) ([]byte, error)
	getStudentProfile(string, string) (string, string, error)
}

type MongoDb struct {
	dbClient *mongo.Client
}

type SqlDb struct {
	dbClient *sql.DB
}

func (self *MongoDb) init(ds string) (err error)  {
	self.dbClient, err = mongo.NewClient(fmt.Sprintf(`mongodb://%s:27017`, ds))
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	err = self.dbClient.Connect(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	err = self.dbClient.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Can't connect to db server.", err)
		return err
	}
	return nil
}

func (self *MongoDb) loadCourses(s *school) error {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	collection := self.dbClient.Database(s.name).Collection("course")
	cur, err := collection.Find(nil, bson.M{})
	if err != nil {
		log.Println(err)
		return err
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		result := course{}
		err := cur.Decode(&result)
		if err != nil {
			log.Println(err)
			return err
		}
		s.courses = append(s.courses,
			NewCourseObj(result.Name, result.Teacher, result.Total, result.Grade))
	}

	return nil
}

func (self *MongoDb) registerCourse(dbName, student, course, teacher string,
	timestamp int64) error {

	collection := self.dbClient.Database(dbName).Collection("register-info")
	_, err := collection.InsertOne(nil, bson.M{
		"student": student,
		"course": course,
		"teacher": teacher,
		"timestamp": timestamp,
	})

	return err
}

func (self *MongoDb) unRegisterCourse(dbName, student, course string) error  {

	collection := self.dbClient.Database(dbName).Collection("register-info")
	cur, err := collection.Find(nil,
		bson.M{"student": student, "course": course},
		options.Find().SetSort(bson.M{"timestamp": -1}).SetLimit(1))
	if err != nil {
		log.Println(err)
		return err
	}

	defer cur.Close(nil)
	for cur.Next(nil) {
		result := registerData{}
		cur.Decode(&result)
		collection.DeleteOne(nil,
			bson.M{"student": result.Student, "timestamp": result.TimeStamp})
		break
	}

	return nil
}

func (self *MongoDb) getRegisterHistory(dbName, student string) ([]byte, error)  {

	registerHistory := struct {
		Data []registerData `json:"data"`
	}{[]registerData{}}

	collection := self.dbClient.Database(dbName).Collection("register-info")
	cur, err := collection.Find(nil,
		bson.M{"student": student},
		options.Find().SetSort(bson.M{"timestamp": -1}))

	if err != nil {
		return nil, err
	}

	defer cur.Close(nil)
	for cur.Next(nil) {
		result := registerData{}
		cur.Decode(&result)
		registerHistory.Data = append(registerHistory.Data, result)
	}

	return json.Marshal(registerHistory)
}

func (self *MongoDb) getStudentProfile(dbName, student string) (string, string, error)  {

	profile := struct {
		Name string `json:"name"`
		Avatar string `json:"avatar"`
	}{}

	collection := self.dbClient.Database(dbName).Collection("profile")
	cur, err := collection.Find(nil, bson.M{"student": student})
	if err != nil {
		return "", "", err
	}

	defer cur.Close(nil)
	for cur.Next(nil) {
		cur.Decode(&profile)
		break
	}
	return profile.Name, profile.Avatar, nil
}

func (self *SqlDb) init(ds string) (err error)  {

	var user = "sa"
	var password = "JESSica1128"
	var database = "mbxsj"

	connString := fmt.Sprintf("server=%s;database=%s;user id=%s;password=%s",
		ds, database, user, password)
	db, err := sql.Open("mssql", connString)
	if err != nil {
		log.Fatal(err)
		return err
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		return err
	}

	self.dbClient = db
	return nil
}

func parseGrade(grade string) []int {
	s := strings.Split(grade, ",")
	if len(s) == 0 {
		return nil
	}

	g := make([]int, len(s))
	for i, v := range s {
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 32)
		g[i] = int(n)
	}
	return g
}

func (self *SqlDb) loadCourses(s *school) error {
	ctx := context.Background()
	sql := fmt.Sprintf("SELECT * FROM course")

	rows, err := self.dbClient.QueryContext(ctx, sql)
	if err != nil {
		log.Println(err)
		return err
	}

	defer rows.Close()

	for rows.Next() {
		result := course{}
		grade := ""
		err := rows.Scan(&result.Name, &result.Teacher, &result.Total, &grade)
		if err != nil {
			log.Println(err)
			return err
		}

		s.courses = append(s.courses,
			NewCourseObj(result.Name, result.Teacher, result.Total, parseGrade(grade)))
	}

	return nil
}

func (self *SqlDb) registerCourse(dbName, student, course, teacher string,
	timestamp int64) error {

	sql := fmt.Sprintf(`INSERT INTO register_info VALUES (N'%s', N'%s', N'%s', %d)`,
		student, course, teacher, timestamp)

	_, err := self.dbClient.Exec(sql)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (self *SqlDb) unRegisterCourse(dbName, student, course string) error {
	ctx := context.Background()
	sql := fmt.Sprintf(`SELECT TOP 1 timestamp FROM register_info WHERE student='%s' AND course='%s' ORDER BY timestamp DESC`,
		student, course)

	rows, err := self.dbClient.QueryContext(ctx, sql)
	if err != nil {
		log.Println(err)
		return err
	}

	defer rows.Close()
	var timestamp int64
	if !rows.Next() {
		err = errors.New("not found")
		return err
	}

	err = rows.Scan(&timestamp)
	if err != nil {
		log.Println(err)
		return err
	}

	sql = fmt.Sprintf(`DELETE FROM register_info WHERE student='%s' AND course='%s' AND timestamp=%d`,
		student, course, timestamp)

	_, err = self.dbClient.Exec(sql)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (self *SqlDb) getRegisterHistory(dbName, student string) ([]byte, error)  {
	registerHistory := struct {
		Data []registerData `json:"data"`
	}{[]registerData{}}

	ctx := context.Background()
	sql := fmt.Sprintf(`SELECT * FROM register_info WHERE student='%s' ORDER BY timestamp DESC`, student)

	rows, err := self.dbClient.QueryContext(ctx, sql)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		result := registerData{}
		err = rows.Scan(&result.Student, &result.Course, &result.Teacher, &result.TimeStamp)
		if err != nil {
			log.Println(err)
		}

		registerHistory.Data = append(registerHistory.Data, result)
	}

	return json.Marshal(registerHistory)
}

func (self *SqlDb) getStudentProfile(dbName, student string) (string, string, error)  {

	ctx := context.Background()
	sql := fmt.Sprintf("SELECT name, avatar FROM profile WHERE student=%s", student)

	rows, err := self.dbClient.QueryContext(ctx, sql)
	if err != nil {
		log.Println(err)
		return "", "", err
	}

	defer rows.Close()

	name, avatar := "", ""
	if !rows.Next() {
		err = errors.New("not found")
	} else {
		err = rows.Scan(&name, &avatar)
		if err != nil {
			log.Println(err)
		}
	}

	return name, avatar, err
}

var _dbs = map[string]database {
	"mongo": &MongoDb{},
	"sql": &SqlDb{},
}

var dbClient = _dbs["sql"]

func initDb(ds string) (err error) {
	return dbClient.init(ds)
}
