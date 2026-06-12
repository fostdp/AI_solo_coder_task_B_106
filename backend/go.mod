module stone-relic-monitor

go 1.21

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.18.0
	github.com/gin-contrib/cors v1.5.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-gota/gota v0.12.0
	github.com/gorilla/websocket v1.5.1
	github.com/prometheus/client_golang v1.18.0
	github.com/spf13/viper v1.18.2
	gonum.org/v1/gonum v0.14.0
	go.uber.org/zap v1.26.0
	stone-relic-monitor/modules/arima v0.0.0
	stone-relic-monitor/modules/robot v0.0.0
	stone-relic-monitor/modules/roughness v0.0.0
	stone-relic-monitor/modules/tsp v0.0.0
	stone-relic-monitor/modules/types v0.0.0
)

replace (
	stone-relic-monitor/modules/arima => ./modules/arima
	stone-relic-monitor/modules/robot => ./modules/robot
	stone-relic-monitor/modules/roughness => ./modules/roughness
	stone-relic-monitor/modules/tsp => ./modules/tsp
	stone-relic-monitor/modules/types => ./modules/types
)

require (
	github.com/ClickHouse/ch-go v1.3.0 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-2522df582 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.16.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-cada74a9e67e // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sagernet/sing v0.4.0 // indirect
	github.com/sagernet/sing-box v1.8.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
