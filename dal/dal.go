package dal

import (
	"net/http"

	"github.com/xsda-pixel/common-infra/errors"
	"github.com/xsda-pixel/common-infra/logs"

	stdErrors "errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var dbErr = errors.NewError(http.StatusBadRequest, errors.NewMsg("db error"))

type WhereOption struct {
	Eq  map[string]any // 等值条件
	Raw *RawWhere      // 原生 SQL 条件
}

type RawWhere struct {
	SQL  string // 原生 SQL 条件
	Args []any  // Raw 参数
}

type JoinOption struct {
	Type  string // LEFT / INNER / RIGHT
	Table string // 关联表
	On    string // 关联条件
}

type ReadRepo[T any] interface {
	FindOne(
		tableName string,
		fields []string,
		where WhereOption,
	) (*T, errors.Error)

	FindOneForUpdate(
		tx *gorm.DB,
		tableName string,
		fields []string,
		where WhereOption,
	) (*T, errors.Error)

	FindMany(
		tableName string,
		fields []string,
		where WhereOption,
		order *string,
		limit *int,
	) ([]*T, errors.Error)

	FindPage(
		tableName string,
		page, limit int,
		fields []string,
		where WhereOption,
		order *string,
	) ([]*T, errors.Error)

	Exists(
		tableName string,
		where WhereOption,
	) (bool, errors.Error)

	Count(
		tableName string,
		where WhereOption,
	) (int64, errors.Error)
}

type WriteRepo[T any] interface {
	CreateOne(
		db *gorm.DB,
		tableName string,
		item *T,
	) errors.Error

	Update(
		db *gorm.DB,
		tableName string,
		where WhereOption,
		updates map[string]any,
	) (int64, errors.Error)

	Delete(
		db *gorm.DB,
		tableName string,
		where WhereOption,
	) (int64, errors.Error)
}

var (
	_ ReadRepo[any]  = (*RepoDB[any])(nil)
	_ WriteRepo[any] = (*RepoDB[any])(nil)
)

func (db *RepoDB[T]) CreateOne(dbs *gorm.DB, tbName string, item *T) errors.Error {
	rs := dbs.Table(tbName).Create(item)
	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return dbErr
	}
	return nil
}

func (db *RepoDB[T]) FindOne(tableName string, fields []string, where WhereOption) (*T, errors.Error) {
	var item T

	rs := db.MySQL.Table(tableName)

	if len(fields) > 0 {
		rs = rs.Select(fields)
	}

	rs = applyWhere(rs, where)

	if err := rs.Take(&item).Error; err != nil {
		if stdErrors.Is(err, gorm.ErrRecordNotFound) {
			// 正常业务分支：没查到
			return nil, nil
		}
		logs.Logger.Error(err)
		return nil, dbErr
	}

	return &item, nil
}

func (db *RepoDB[T]) FindOneForUpdate(
	tx *gorm.DB,
	tableName string,
	fields []string,
	where WhereOption,
) (*T, errors.Error) {
	var item T

	rs := tx.Table(tableName)

	if len(fields) > 0 {
		rs = rs.Select(fields)
	}

	rs = applyWhere(rs, where)

	rs = rs.Clauses(clause.Locking{
		Strength: "UPDATE", // FOR UPDATE
	})

	if err := rs.Take(&item).Error; err != nil {
		if stdErrors.Is(err, gorm.ErrRecordNotFound) {
			// 正常业务分支：没查到
			return nil, nil
		}
		logs.Logger.Error(err)
		return nil, dbErr
	}

	return &item, nil
}

func (db *RepoDB[T]) FindMany(tableName string, fields []string, where WhereOption, order *string, limit *int) ([]*T, errors.Error) {
	var list []*T

	rs := db.MySQL.Table(tableName)

	if len(fields) > 0 {
		rs = rs.Select(fields)
	}

	rs = applyWhere(rs, where)

	if order != nil && *order != "" {
		rs = rs.Order(*order)
	}

	if limit != nil && *limit > 0 {
		rs = rs.Limit(*limit)
	}

	rs = rs.Find(&list)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return nil, dbErr
	}

	return list, nil
}

func (db *RepoDB[T]) FindManyWithGroupBy(
	tableName string,
	fields []string,
	groupBy string,
	where WhereOption,
	order *string,
	limit *int,
) ([]*T, errors.Error) {
	var list []*T

	rs := db.MySQL.Table(tableName)

	if len(fields) > 0 {
		rs = rs.Select(fields)
	}

	if groupBy != "" {
		rs = rs.Group(groupBy)
	}

	rs = applyWhere(rs, where)

	if order != nil && *order != "" {
		rs = rs.Order(*order)
	}

	if limit != nil && *limit > 0 {
		rs = rs.Limit(*limit)
	}

	rs = rs.Find(&list)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return nil, dbErr
	}

	return list, nil
}

func (db *RepoDB[T]) FindPage(
	tableName string,
	page, limit int,
	fields []string,
	where WhereOption,
	order *string,
) ([]*T, errors.Error) {
	var (
		list []*T
	)

	if page < 1 || limit < 1 {
		return list, nil
	}

	rs := db.MySQL.Table(tableName)

	if len(fields) > 0 {
		rs = rs.Select(fields)
	}

	rs = applyWhere(rs, where)

	if order != nil && *order != "" {
		rs = rs.Order(*order)
	}

	rs = rs.Limit(limit).Offset((page - 1) * limit).Find(&list)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return nil, dbErr
	}

	return list, nil
}

func (db *RepoDB[T]) Count(
	tableName string,
	where WhereOption,
) (int64, errors.Error) {

	var count int64

	rs := db.MySQL.Table(tableName)

	rs = applyWhere(rs, where)

	rs = rs.Count(&count)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return 0, dbErr
	}

	return count, nil
}

func (db *RepoDB[T]) SumInt64(
	tableName string,
	expr string,
	where WhereOption,
) (int64, errors.Error) {
	var result int64

	rs := db.MySQL.Table(tableName).
		Select(expr)

	rs = applyWhere(rs, where)

	if err := rs.Scan(&result).Error; err != nil {
		logs.Logger.Error(err)
		return 0, dbErr
	}

	return result, nil
}

func (db *RepoDB[T]) Exists(
	tableName string,
	where WhereOption,
) (bool, errors.Error) {
	var tmp int

	rs := db.MySQL.Table(tableName).Select("1")

	rs = applyWhere(rs, where)

	rs = rs.Limit(1).Scan(&tmp)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return false, dbErr
	}

	return rs.RowsAffected > 0, nil
}

func (db *RepoDB[T]) Update(
	dbs *gorm.DB,
	tableName string,
	where WhereOption,
	updates map[string]any,
) (int64, errors.Error) {
	if len(updates) == 0 {
		return 0, nil
	}

	rs := dbs.Table(tableName)

	rs = applyWhere(rs, where)

	rs = rs.Updates(updates)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return 0, dbErr
	}

	return rs.RowsAffected, nil
}

func (db *RepoDB[T]) Delete(
	dbs *gorm.DB,
	tableName string,
	where WhereOption,
) (int64, errors.Error) {
	rs := dbs.Table(tableName)

	if len(where.Eq) == 0 && where.Raw == nil {
		return 0, errors.NewError(http.StatusBadRequest, errors.NewMsg("delete without where is forbidden")) // 防止误删
	}

	rs = applyWhere(rs, where)

	rs = rs.Delete(nil)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return 0, dbErr
	}

	return rs.RowsAffected, nil
}

func (db *RepoDB[T]) FindManyWithJoin(
	tableName string,
	fields []string,
	joins []JoinOption,
	where WhereOption,
	order *string,
	limit *int,
) ([]*T, errors.Error) {
	var list []*T

	rs := db.MySQL.Table(tableName)

	if len(fields) > 0 {
		rs = rs.Select(fields)
	}

	rs = applyJoins(rs, joins)

	rs = applyWhere(rs, where)

	if order != nil && *order != "" {
		rs = rs.Order(*order)
	}

	if limit != nil && *limit > 0 {
		rs = rs.Limit(*limit)
	}

	rs = rs.Find(&list)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return nil, dbErr
	}

	return list, nil
}

func (db *RepoDB[T]) FindPageWithJoin(
	tableName string,
	page, limit int,
	fields []string,
	joins []JoinOption,
	where WhereOption,
	order *string,
) ([]*T, errors.Error) {
	var list []*T

	if page < 1 || limit < 1 {
		return list, nil
	}

	rs := db.MySQL.Table(tableName)

	if len(fields) > 0 {
		rs = rs.Select(fields)
	}

	rs = applyJoins(rs, joins)

	rs = applyWhere(rs, where)

	if order != nil && *order != "" {
		rs = rs.Order(*order)
	}

	rs = rs.Limit(limit).Offset((page - 1) * limit).Find(&list)

	if rs.Error != nil {
		logs.Logger.Error(rs.Error)
		return nil, dbErr
	}

	return list, nil
}

func applyWhere(rs *gorm.DB, w WhereOption) *gorm.DB {
	if len(w.Eq) > 0 {
		rs = rs.Where(w.Eq)
	}
	if w.Raw != nil {
		rs = rs.Where(w.Raw.SQL, w.Raw.Args...)
	}
	return rs
}

func applyJoins(rs *gorm.DB, joins []JoinOption) *gorm.DB {
	for _, j := range joins {
		joinType := "LEFT JOIN"
		if j.Type != "" {
			joinType = j.Type + " JOIN"
		}
		rs = rs.Joins(joinType + " " + j.Table + " ON " + j.On)
	}
	return rs
}

/*
Example:

Target:

SELECT u.id, u.name, o.amount
FROM user u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.status != 1 AND u.id IN (1,2,3)
LIMIT 10

Use:
users, err := repo.FindManyWithJoin(
	"user u",
	[]string{
		"u.id",
		"u.name",
		"o.amount",
	},
	[]JoinOption{
		{
			Type:  "LEFT",
			Table: "orders o",
			On:    "o.user_id = u.id",
		},
	},
	WhereOption{
		Raw: &RawWhere{SQL: "u.status != ? AND u.id IN ?", Args: []any{1, []int{1, 2, 3}}},
	},
	ptr(""),
	ptr(10),
)


Target:

SELECT u.id AS user_id, u.name, o.id AS order_id, o.amount
FROM user u
LEFT JOIN orders o ON o.user_id = u.id
WHERE u.status != 1
  AND u.id IN (1,2,3,4)
ORDER BY u.id DESC
LIMIT 10 OFFSET 0


Use:
repo := NewRepoDB[UserOrderDTO](db)

list, err := repo.FindPageWithJoin(
	"user u",
	1,
	10,
	[]string{
		"u.id AS user_id",
		"u.name",
		"o.id AS order_id",
		"o.amount",
	},
	[]JoinOption{
		{
			Type:  "LEFT",
			Table: "orders o",
			On:    "o.user_id = u.id",
		},
	},
	WhereOption{
		Raw: &RawWhere{SQL: "u.status != ? AND u.id IN ?", Args: []any{1, []int{1, 2, 3, 4}}},
	},
	ptr("u.id DESC"),
)
*/
