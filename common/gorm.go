package common

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"
	"unicode"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SQLDialect ..
const SQLDialect = "mysql"

// OpenGorm 初始化gorm，开启日志，并在create/query时自动处理BigIn类型，注意更新BigInt字段时仍需使用gorm.Expr
func OpenGorm(connectStr string) (*gorm.DB, error) {
	db, err := gorm.Open(SQLDialect, connectStr)
	if err != nil {
		return db, errors.WithStack(err)
	}
	db.SetLogger(&GormLoger{})
	db.LogMode(true)
	db.Callback().Create().Before("gorm:create").Register("big_int:before_create", func(scope *gorm.Scope) {
		for _, f := range scope.Fields() {
			v := f.Field.Interface()
			if bi, ok := v.(BigInt); ok {
				f.IsNormal = true
				f.Field = reflect.ValueOf(gorm.Expr("cast(? AS DECIMAL(65,0))", bi.String()))
			}
		}
	})
	return db, nil
}

// StopDB stop gorm db
func StopDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	err := db.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// WithTransaction 简单封装事务处理
func WithTransaction(db *gorm.DB, f func(db *gorm.DB) error) error {
	dbTx := db.Begin()
	if dbTx.Error != nil {
		err := errors.Wrap(dbTx.Error, "failed to open tx")
		logrus.Error(err)
		return err
	}
	if err := f(dbTx); err != nil {
		rollBackErr := dbTx.Rollback().Error
		if rollBackErr != nil {
			logrus.Error(rollBackErr)
		}
		return err
	}
	if err := dbTx.Commit().Error; err != nil {
		err = errors.Wrap(err, "failed to commit tx")
		logrus.Error(err)
		rollBackErr := dbTx.Rollback().Error
		if rollBackErr != nil {
			logrus.Error(rollBackErr)
		}
		return err
	}
	return nil
}

// GormLoger ...
type GormLoger struct {
}

// Print ..
func (l *GormLoger) Print(v ...interface{}) {
	Adapt(v...)
}

var (
	sqlRegexp                = regexp.MustCompile(`\?`)
	numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
)

// DebugRawSQLExcution ..
func DebugRawSQLExcution(sqlStatement string, vars []interface{}, timeUsed time.Duration, rowsAffected int64) {
	Adapt([]interface{}{"sql", "", timeUsed, sqlStatement, vars, rowsAffected}...)
}

// Adapt modify from gorm.LogFormatter
// 自行拼装的RawSql如果要用同样的格式打印需要提供[]interface{}{"sql","",time.Duration indicate your cost, prepared sql statement with '?', []interface{} your vars to fill the statement, int64 rowsAffected}
func Adapt(values ...interface{}) {
	if len(values) > 1 {
		var (
			sql             string
			formattedValues []string
			level           = values[0]
		)

		if level == "sql" {
			// duration
			timeUsed := fmt.Sprintf("%.2fms", float64(values[2].(time.Duration).Nanoseconds()/1e4)/100.0)
			// sql

			for _, value := range values[4].([]interface{}) {
				indirectValue := reflect.Indirect(reflect.ValueOf(value))
				if indirectValue.IsValid() {
					value = indirectValue.Interface()
					if t, ok := value.(time.Time); ok {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
					} else if b, ok := value.([]byte); ok {
						if str := string(b); isPrintable(str) {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
						} else {
							formattedValues = append(formattedValues, "'<binary>'")
						}
					} else if r, ok := value.(driver.Valuer); ok {
						if value, err := r.Value(); err == nil && value != nil {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
						} else {
							formattedValues = append(formattedValues, "NULL")
						}
					} else {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
					}
				} else {
					formattedValues = append(formattedValues, "NULL")
				}
			}

			// differentiate between $n placeholders or else treat like ?
			if numericPlaceHolderRegexp.MatchString(values[3].(string)) {
				sql = values[3].(string)
				for index, value := range formattedValues {
					placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
					sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
				}
			} else {
				formattedValuesLength := len(formattedValues)
				for index, value := range sqlRegexp.Split(values[3].(string), -1) {
					sql += value
					if index < formattedValuesLength {
						sql += formattedValues[index]
					}
				}
			}

			logrus.WithField("cost", timeUsed).WithField("rowsAffected", strconv.FormatInt(values[5].(int64), 10)).Debug(sql)
		} else {
			logrus.Info(values...)
		}
	}
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
