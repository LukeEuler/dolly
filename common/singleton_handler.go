package common

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/LukeEuler/dolly/log"
)

/*
SingletonHandler 以`mysql`为基础的单实例实现
会自动在数据库中创建表 uuid_singleton_lock
*/
type SingletonHandler struct {
	db                               *sql.DB
	name, uuid                       string
	refreshInterval, expiredInterval int
}

func NewSingletonHandler(name, source string, refreshInterval, expiredInterval int) (s *SingletonHandler, err error) {
	s = &SingletonHandler{
		name:            name,
		uuid:            uuid.NewV4().String(),
		refreshInterval: refreshInterval,
		expiredInterval: expiredInterval,
	}

	log.Entry.
		WithField("tags", "initialization").
		WithField("name", s.name).
		WithField("uuid", s.uuid).
		WithField("refresh interval(s)", s.refreshInterval).
		WithField("expired interval(s)", s.expiredInterval).
		Info()

	driver := "mysql"
	s.db, err = sql.Open(driver, source)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, err = s.db.Exec(`
	CREATE TABLE IF NOT EXISTS uuid_singleton_lock (
		name VARCHAR(255) NOT NULL COMMENT 'format <platform>_<chain>_<component>, example pro_btc_updater',
		uuid VARCHAR(255) NOT NULL,
		last_update_time BIGINT(13) UNSIGNED NOT NULL,
		PRIMARY KEY (name),
		UNIQUE idx_uuid (uuid)
	) ENGINE = InnoDB
	  DEFAULT CHARSET = utf8mb4 DEFAULT COLLATE 'utf8mb4_bin'`)
	if err != nil {
		_ = s.db.Close()
		return nil, errors.WithStack(err)
	}

	item := &struct {
		uuid           string
		lastUpdateTime int64
	}{}
	var now int64
	err = s.db.QueryRow("SELECT uuid, last_update_time, ROUND(UNIX_TIMESTAMP(CURTIME(4)) * 1000) FROM uuid_singleton_lock WHERE name = ?", s.name).
		Scan(&item.uuid, &item.lastUpdateTime, &now)
	if err != nil && err != sql.ErrNoRows {
		_ = s.db.Close()
		return nil, errors.WithStack(err)
	}

	if err == sql.ErrNoRows {
		_, err = s.db.Exec("INSERT uuid_singleton_lock (name, uuid, last_update_time) VALUE (?, ?, ROUND(UNIX_TIMESTAMP(CURTIME(4)) * 1000))", s.name, s.uuid)
		if err != nil {
			_ = s.db.Close()
			return nil, errors.WithStack(err)
		}
	} else {
		if item.uuid == s.uuid {
			_ = s.db.Close()
			return nil, errors.Errorf("got the same uuid[%s], name %s", item.uuid, s.name)
		}
		if now-item.lastUpdateTime <= int64(s.expiredInterval)*1000 {
			_ = s.db.Close()
			return nil, errors.Errorf("can not regist %s at %d while last update time is %d", s.name, now, item.lastUpdateTime)
		}
		_, err = s.db.Exec("UPDATE uuid_singleton_lock SET uuid = ?, last_update_time = ROUND(UNIX_TIMESTAMP(CURTIME(4)) * 1000) WHERE name = ?", s.uuid, s.name)
		if err != nil {
			_ = s.db.Close()
			return nil, errors.WithStack(err)
		}
	}

	return s, nil
}

// Loop 维持心跳
func (s *SingletonHandler) Loop(shutdown chan struct{}) {
	interval := time.Duration(s.refreshInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-shutdown:
			log.Entry.Debug("stop singletonHandler")
			_, err := s.db.Exec("UPDATE uuid_singleton_lock SET last_update_time = 0 WHERE name = ? AND uuid = ?", s.name, s.uuid)
			if err != nil {
				log.Entry.Error(err)
			}
			err = s.db.Close()
			if err != nil {
				log.Entry.Error(err)
			}
			return
		case <-ticker.C:
			res, err := s.db.Exec("UPDATE uuid_singleton_lock SET last_update_time = ROUND(UNIX_TIMESTAMP(CURTIME(4)) * 1000) WHERE name = ? AND uuid = ?", s.name, s.uuid)
			if err != nil {
				log.Entry.Error(err)
				close(shutdown)
				err = s.db.Close()
				if err != nil {
					log.Entry.Error(err)
				}
				return
			}
			affected, err := res.RowsAffected()
			if err != nil {
				log.Entry.Error(err)
				close(shutdown)
				err = s.db.Close()
				if err != nil {
					log.Entry.Error(err)
				}
				return
			}
			if affected != 1 {
				err = fmt.Errorf("affected %d: UPDATE uuid_singleton_lock SET last_update_time = ROUND(UNIX_TIMESTAMP(CURTIME(4)) * 1000) WHERE name = '%s' AND uuid = '%s'", affected, s.name, s.uuid)
				log.Entry.Error(err)
				close(shutdown)
				err = s.db.Close()
				if err != nil {
					log.Entry.Error(err)
				}
				return
			}
		}
	}
}
