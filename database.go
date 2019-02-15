package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"log"
	"time"
)

type database interface {
	init(string) error
	loadCourses(*school) error
	registerCourse(string, string, string, string, int64) error
	unRegisterCourse(string, string, string)
	getRegisterHistory(string, string) ([]byte, error)
}

type MongoDb struct {
	dbClient *mongo.Client
}

type SqlDb struct {
}

func (self *MongoDb) init(ds string) (err error)  {
	self.dbClient, err = mongo.NewClient(fmt.Sprintf(`mongodb://%s:27017`, ds))
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	err = self.dbClient.Connect(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return err
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

func (self *MongoDb) unRegisterCourse(dbName, student, course string)  {

	collection := self.dbClient.Database(dbName).Collection("register-info")
	cur, err := collection.Find(nil,
		bson.M{"student": student, "course": course},
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

func (self *SqlDb) init(ds string) (err error)  {

	return nil
}

func (self *SqlDb) loadCourses(*school) error {
	return nil
}

func (self *SqlDb) registerCourse(dbName, student, course, teacher string,
	timestamp int64) error {

		return nil
}

func (self *SqlDb) unRegisterCourse(dbName, student, course string)  {

}

func (self *SqlDb) getRegisterHistory(dbName, student string) ([]byte, error)  {
	return nil, nil
}

var _dbs = map[string]database {
	"mongo": &MongoDb{},
	"sql": &SqlDb{},
}

var dbClient = _dbs["mongo"]

func initDb(ds string) (err error) {
	return dbClient.init(ds)
}
