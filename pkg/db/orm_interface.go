package db

import (
	"xorm.io/xorm"
)

type ormInterface struct {
	ORM *xorm.EngineInterface
}

var instance ormInterface

func init() {}
