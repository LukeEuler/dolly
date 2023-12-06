package common

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"github.com/LukeEuler/dolly/log"
)

// OpenGorm 初始化gorm，开启日志，并在create/query时自动处理BigIn类型，注意更新BigInt字段时仍需使用gorm.Expr
func OpenGorm(connectStr string, fileds logrus.Fields) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(connectStr), &gorm.Config{
		Logger: NewGormLogger(fileds),
	})
	if err != nil {
		return db, errors.Wrapf(err, "connect db %s", hideUserPass(connectStr))
	}
	return db, nil
}

func StopDB(db *gorm.DB) error {
	defer TimeConsume(time.Now())
	if db == nil {
		return nil
	}
	innerDB, _ := db.DB()
	if innerDB != nil {
		err := innerDB.Close()
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func StopDBV2(db *gorm.DB) {
	defer TimeConsume(time.Now())
	if db == nil {
		return
	}
	innerDB, _ := db.DB()
	if innerDB != nil {
		err := innerDB.Close()
		if err != nil {
			log.Entry.Error(err)
		}
	}
}

func hideUserPass(connectStr string) string {
	conn := strings.Split(connectStr, "@tcp")
	if len(conn) > 1 {
		return conn[1]
	}
	return connectStr
}

// WithTransaction 简单封装事务处理
func WithTransaction(db *gorm.DB, f func(db *gorm.DB) error) error {
	dbTx := db.Begin()
	if dbTx.Error != nil {
		err := errors.Wrap(dbTx.Error, "failed to open tx")
		log.Entry.Error(err)
		return err
	}
	if err := f(dbTx); err != nil {
		rollBackErr := dbTx.Rollback().Error
		if rollBackErr != nil {
			log.Entry.Error(rollBackErr)
		}
		return err
	}
	if err := dbTx.Commit().Error; err != nil {
		err = errors.Wrap(err, "failed to commit tx")
		log.Entry.Error(err)
		rollBackErr := dbTx.Rollback().Error
		if rollBackErr != nil {
			log.Entry.Error(rollBackErr)
		}
		return err
	}
	return nil
}

// HandleDBTX 封装对事务的处理
func HandleDBTX(db *gorm.DB, f func(*gorm.DB, Context) error, ctx Context) error {
	defer TimeConsume(time.Now())
	dbTx := db.Begin()
	var err error
	defer RecoverV2(dbTx, &err)

	err = dbTx.Error
	if err != nil {
		log.Entry.Error(errors.Wrap(err, "new procedure"))
		return err
	}

	err = f(dbTx, ctx)
	if err != nil {
		SilentlyRollback(dbTx)
		return err
	}

	err = dbTx.Commit().Error
	if err != nil {
		log.Entry.Error(errors.Wrap(err, "roll back procedure"))
		SilentlyRollback(dbTx)
	}
	return err
}

func HandleDBTXWithLock(db *gorm.DB, mutex *sync.RWMutex, f func(*gorm.DB, Context) error, ctx Context) error {
	mutex.Lock()
	defer mutex.Unlock()
	return HandleDBTX(db, f, ctx)
}

// SilentlyRollback 回滚 dbTx，并打印回滚错误
func SilentlyRollback(dbTx *gorm.DB) {
	rollBackErr := dbTx.Rollback().Error
	if rollBackErr != nil {
		log.Entry.Error(errors.Wrap(rollBackErr, "roll back procedure"))
	} else {
		log.Entry.Info("roolback complete")
	}
}

type GormLogger struct {
	IgnoreRecordNotFoundError bool
	SlowThreshold             time.Duration
	fileds                    logrus.Fields
}

func NewGormLogger(fileds logrus.Fields) logger.Interface {
	return &GormLogger{
		IgnoreRecordNotFoundError: true,
		SlowThreshold:             200 * time.Millisecond,
		fileds:                    fileds,
	}
}

func (l *GormLogger) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	log.Entry.
		WithField("gorm", "others").
		WithFields(l.fileds).
		Infof(msg, append([]any{utils.FileWithLineNum()}, data...)...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	log.Entry.
		WithField("gorm", "others").
		WithFields(l.fileds).
		Warnf(msg, append([]any{utils.FileWithLineNum()}, data...)...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	log.Entry.
		WithField("gorm", "others").
		WithFields(l.fileds).
		Errorf(msg, append([]any{utils.FileWithLineNum()}, data...)...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	rawsStr := "-"
	if rows != -1 {
		rawsStr = strconv.Itoa(int(rows))
	}

	entry := log.Entry.WithFields(logrus.Fields{
		"gorm":   "sql",
		"source": utils.FileWithLineNum(),
		"cost":   fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6),
		"sql":    sql,
		"rows":   rawsStr,
	}).
		WithFields(l.fileds)
	switch {
	case err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		entry.Error(err)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		entry.Warn(slowLog)
	default:
		entry.Debug()
	}
}
