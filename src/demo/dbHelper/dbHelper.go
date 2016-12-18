package dbHelper

import mgo "gopkg.in/mgo.v2"

// CollectionInfo Collection信息
type CollectionInfo struct {
	Name        string
	RecordCount int
}

// MongoDBHelper MongonDB操作助手
type MongoDBHelper interface {
	Open(svrAddr, databaseName string) bool
	Close()
	Collections() ([]CollectionInfo, bool)
	FetchCollection(name string) (*mgo.Collection, bool)
}

type impl struct {
	session  *mgo.Session
	database *mgo.Database
}

// NewDBHelper 新建DBHelper
func NewDBHelper() MongoDBHelper {
	s := &impl{session: nil, database: nil}

	return s
}

func (s *impl) Open(svrAddr, dbName string) bool {
	var err error
	s.session, err = mgo.Dial(svrAddr)
	if err != nil {
		return false
	}

	s.database = s.session.DB(dbName)

	return true
}

func (s *impl) Close() {
	if s.session == nil {
		return
	}

	s.session.Close()
}

func (s *impl) Collections() ([]CollectionInfo, bool) {
	collections := []CollectionInfo{}
	collectNames := []string{}
	if s.database == nil {
		return collections, false
	}

	collectNames, err := s.database.CollectionNames()
	if err != nil {
		return collections, false
	}

	for _, name := range collectNames {
		info := CollectionInfo{}
		info.Name = name
		c := s.database.C(name)
		info.RecordCount, _ = c.Count()

		collections = append(collections, info)
	}

	return collections, true
}

func (s *impl) FetchCollection(name string) (*mgo.Collection, bool) {
	if s.database == nil {
		return nil, false
	}

	c := s.database.C(name)
	return c, c != nil
}
