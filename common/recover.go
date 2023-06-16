package common

import (
	"fmt"
	"runtime/debug"

	"gorm.io/gorm"

	"github.com/LukeEuler/dolly/log"
)

/*
Recover 通用协程 panic 的记录方法
请将该函数置于 主进程 或 协程的开始处

example：

	func main() {
		defer common.Recover()
		...
	}
*/
func Recover() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%v\nstacktrace from panic: %s", r, string(debug.Stack()))
		log.Entry.Error(err)
		// 以防 log 失效
		fmt.Printf("\n------------- avoid log failure -------------\n%s\n---------------------------------------------\n",
			err)
	}
}

/*
RecoverV2 通用协程 panic 的记录方法

当下层程序 panic, 则 rollback dbTx，且将错误赋值 err
else return false

example：

	func aFunc(...) (err error) {
		dbTx := db.Begin() // db *gorm.DB
		defer common.RecoverV2(dbTx, &err)
		...
	}
*/
func RecoverV2(dbTx *gorm.DB, err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%v\nstacktrace from panic: %s", r, string(debug.Stack()))
		log.Entry.Error(*err)
		// 以防 log 失效
		fmt.Printf("\n------------- avoid log failure -------------\n%s\n---------------------------------------------\n",
			*err)
		SilentlyRollback(dbTx)
		return
	}
}
