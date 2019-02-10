package main

import (
	"context"
	"fmt"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"log"
	"time"
)

var dbClient *mongo.Client

func initDb(ds string) (err error) {
	dbClient, err = mongo.NewClient(fmt.Sprintf(`mongodb://%s:27017`, ds))
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	err = dbClient.Connect(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	go dbRoutine()

	return err
}

func (s *school) loadCourses() error {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	collection := dbClient.Database(s.name).Collection("course")
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

type chanHandler interface {
	handle()
}

type chanRegister struct {
	db string
	data registerData
}

func (self *chanRegister) handle() {
	collection := dbClient.Database(self.db).Collection("register-info")
	_, err := collection.InsertOne(nil, bson.M{
		"student": self.data.Student,
		"course": self.data.Course,
		"teacher": self.data.Teacher,
		"timestamp": self.data.TimeStamp,
	})
	if err != nil {
		log.Println(err)
	}
}
type chanUnRegister struct {
	db string
	student string
	course string
}

func (self *chanUnRegister) handle() {
	collection := dbClient.Database(self.db).Collection("register-info")
	cur, err := collection.Find(nil,
		bson.M{"student": self.student, "course": self.course},
		options.Find().SetSort(bson.M{"timestamp": -1}).SetLimit(1))
	if err != nil {
		log.Println(err)
		return
	}

	defer cur.Close(nil)
	for cur.Next(nil) {
		result := registerData{}
		cur.Decode(&result)
		collection.DeleteOne(nil,
			bson.M{"student": result.Student, "timestamp": result.TimeStamp})
		break
	}
}

type registerData struct {
	Student string `json:"student"`
	Course string `json:"course"`
	Teacher string `json:"teacher"`
	TimeStamp int64 `json:"timestamp"`
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
		db: s.name,
		data: registerData{student, c.Name, c.Teacher, time.Now().Unix()},
	}
}

func (s *school) unRegisterDb(student, course string) {
	dbChannel <- &chanUnRegister{s.name, student, course}
}