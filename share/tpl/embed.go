package tpl

import (
	"embed"
	"io/fs"
)

// 配置表 JSON 内嵌进二进制（gen-config.bat 会同步拷贝到本目录），部署不依赖运行目录。
//
//go:embed data/*.json
var dataFS embed.FS

// loadRaw 读取内嵌的配置表 JSON。
func loadRaw(file string) ([]byte, error) {
	return fs.ReadFile(dataFS, "data/"+file+".json")
}
