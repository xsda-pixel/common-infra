package dal

import (
	rds "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type DBS struct {
	MySQL *gorm.DB
	RDS   *rds.Client
}

type RepoDB[T any] struct {
	*DBS
}

func NewDB(db *gorm.DB, rdsClient *rds.Client) *DBS {
	return &DBS{
		MySQL: db,
		RDS:   rdsClient,
	}
}
