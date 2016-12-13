package dbHelper

import mgo "gopkg.in/mgo.v2"

// MongoDBHelper MongonDB操作助手
type MongoDBHelper interface {
	Open(svrAddr, databaseName string) bool
	Close()
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

func (s *impl) CollectionName() ([]string, bool) {
	collectNames := []string{}
	if s.database == nil {
		return collectNames, false
	}

	collectNames, err := s.database.CollectionNames()
	if err != nil {
		return collectNames, false
	}

	return collectNames, true
}

func (s *impl) FetchCollection(name string) (*mgo.Collection, bool) {
	if s.database == nil {
		return nil, false
	}

	c := s.database.C(name)
	c.Count()
	return c, c != nil
}
