module github.com/garethgeorge/backrest

go 1.24

// Pin dependencies to specific versions
replace (
	modernc.org/libc => modernc.org/libc v1.55.3
	modernc.org/mathutil => modernc.org/mathutil v1.6.0
	modernc.org/memory => modernc.org/memory v1.8.0
	modernc.org/sqlite => modernc.org/sqlite v1.33.1
	zombiezen.com/go/sqlite => zombiezen.com/go/sqlite v1.4.0
)

require (
	al.essio.dev/pkg/shellescape v1.6.0
	connectrpc.com/connect v1.18.1
	github.com/containrrr/shoutrrr v0.8.0
	github.com/djherbis/buffer v1.2.0
	github.com/djherbis/nio/v3 v3.0.1
	github.com/getlantern/systray v1.2.2
	github.com/gitploy-io/cronexpr v0.2.2
	github.com/gofrs/flock v0.12.1
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/go-cmp v0.7.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/mattn/go-colorable v0.1.14
	github.com/natefinch/atomic v1.0.1
	github.com/ncruces/zenity v0.10.14
	github.com/prometheus/client_golang v1.22.0
	github.com/stretchr/testify v1.10.0
	github.com/vearutop/statigz v1.5.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.38.0
	golang.org/x/net v0.40.0
	golang.org/x/sync v0.14.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250505200425-f936aa4a68b2
	google.golang.org/grpc v1.72.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	zombiezen.com/go/sqlite v1.4.0
)

require (
	github.com/akavel/rsrc v0.10.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/jsmin v1.0.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/getlantern/context v0.0.0-20220418194847-3d5e7a086201 // indirect
	github.com/getlantern/errors v1.0.4 // indirect
	github.com/getlantern/golog v0.0.0-20230503153817-8e72de7e0a65 // indirect
	github.com/getlantern/hex v0.0.0-20220104173244-ad7e4b9194dc // indirect
	github.com/getlantern/hidden v0.0.0-20220104173330-f221c5a24770 // indirect
	github.com/getlantern/ops v0.0.0-20231025133620-f368ab734534 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/josephspurrier/goversioninfo v1.5.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.63.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/randall77/makefat v0.0.0-20210315173500-7ddd0e42c844 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	golang.org/x/image v0.27.0 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250505200425-f936aa4a68b2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.65.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.10.0 // indirect
	modernc.org/sqlite v1.37.0 // indirect
)
