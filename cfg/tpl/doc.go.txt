// Package cfg 配置表运行时包：加载 Luban 导出的 JSON，按 id 查询。
//
// 目录约定（share/cfg）：
//
//	item.go / weapon.go / tables.go …              生成后按表合并，勿手改
//	cfg.go / doc.go / cfg_smoke_test.go            手写，源文件在 cfg/tpl，生成后自动拷回
//	data/*.json                                    Luban 生成的 JSON，go:embed 内嵌进二进制
//
// 使用：
//
//	if err := cfg.Load(); err != nil { … }
//	w := cfg.T().TbWeapon.Get(2)
//
// 生成：deploy\scripts\gen-config.bat
//
// 新增表命名规范（cfg/excel/__tables__.xlsx 登记）：
//
//	full_name   表类名，Tb 前缀、不带模块前缀，如 TbWeapon
//	value_type  记录结构名，裸类型名，如 Weapon
//	output      数据文件名（不含扩展名），填模块名，如 weapon
//
// 生成结果：Weapon（记录结构）+ TbWeapon（表类），
// 文件 weapon.go / tb_weapon.go / tables.go，数据 data/weapon.json。
package cfg
