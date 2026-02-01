module github.com/xsda-pixel/common-infra

go 1.21

require (
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/sirupsen/logrus v1.9.3
	// 使用旧一点的稳定版本，兼容 Go 1.21
	golang.org/x/sync v0.9.0
	golang.org/x/time v0.5.0
)

require (
	github.com/redis/go-redis/v9 v9.17.3
	gorm.io/gorm v1.31.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/lestrrat-go/strftime v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.20.0 // indirect
)

// 保持这个 replace 能够防止 x/sys 自动升级炸掉环境
replace golang.org/x/sys => golang.org/x/sys v0.18.0
