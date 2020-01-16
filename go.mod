module github.com/iotexproject/iotex-analytics

go 1.12

require (
	github.com/99designs/gqlgen v0.8.3
	github.com/agnivade/levenshtein v1.0.2 // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/ethereum/go-ethereum v1.8.27
	github.com/go-sql-driver/mysql v1.4.1
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/iotexproject/go-pkgs v0.1.1
	github.com/iotexproject/iotex-address v0.2.1
	github.com/iotexproject/iotex-core v0.8.1-0.20191007232750-b79fb5c7ebaa
	github.com/iotexproject/iotex-election v0.2.7-0.20191008203349-58450eac6656
	github.com/iotexproject/iotex-proto v0.2.5
	github.com/lzxm160/grpc-rest v0.0.0-20190822130803-a13cc3e2237e // indirect
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/testify v1.3.0
	github.com/vektah/gqlparser v1.1.2
	go.uber.org/zap v1.10.0
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	google.golang.org/grpc v1.23.0
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/ethereum/go-ethereum => github.com/iotexproject/go-ethereum v0.2.0

exclude github.com/dgraph-io/badger v2.0.0-rc.2+incompatible

exclude github.com/dgraph-io/badger v2.0.0-rc2+incompatible

exclude github.com/ipfs/go-ds-badger v0.0.3
