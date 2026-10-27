package global

import (
	"github.com/dobyte/due/v2/log"

	"server-cluster/internal/svc"
	"server-cluster/mesh/admin/app/repo"
	"server-cluster/share/model"
)

// adminTables 管理后台全部表模型。
var adminTables = []any{
	&model.SysUser{},
	&model.SysRole{},
	&model.SysMenu{},
	&model.SysApi{},
	&model.SysDictType{},
	&model.SysDictData{},
	&model.SysConfig{},
	&model.SysLoginLog{},
	&model.SysOperaLog{},
	&model.CasbinRule{},
}

// Init 初始化应用：svc（DB/Redis/雪花ID/JWT）→ 自动建表 → 数据访问句柄 → 种子数据。
// 须在注册路由前调用。
func Init() error {
	if err := svc.Init(); err != nil {
		return err
	}
	gdb, err := svc.Ctx().DB()
	if err != nil {
		log.Warnf("admin: mysql not configured, admin apis disabled: %v", err)
		gdb = nil
	}
	if err := svc.AutoMigrate(gdb, adminTables...); err != nil {
		return err
	}
	repo.Init(gdb)
	if gdb != nil {
		if err := seed(gdb); err != nil {
			return err
		}
	}
	return nil
}

// Destroy 进程退出后释放 DB / Redis 等资源。
func Destroy() {
	_ = svc.Close()
}
