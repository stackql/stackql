module github.com/stackql/stackql

go 1.18

require (
	github.com/aws/aws-sdk-go v1.28.8
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/fatih/color v1.13.0
	github.com/getkin/kin-openapi v0.88.0
	github.com/google/go-jsonnet v0.17.0
	github.com/jackc/pgtype v1.10.0
	github.com/jeroenrinzema/psql-wire v0.0.1-stackqlalpha3
	github.com/lib/pq v1.10.4
	github.com/magiconair/properties v1.8.6
	github.com/olekukonko/tablewriter v0.0.0-20180130162743-b8a9be070da4
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stackql/go-openapistackql v0.0.10-alpha18
	github.com/stackql/go-sqlite3 v0.0.1-stackqlrc01
	github.com/stackql/go-suffix-map v0.0.1-alpha01
	go.uber.org/zap v1.21.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f
	gonum.org/v1/gonum v0.11.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	vitess.io/vitess v0.0.11-alpha04
)

require (
	cloud.google.com/go v0.99.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/antchfx/xmlquery v1.3.10 // indirect
	github.com/antchfx/xpath v1.2.0 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stackql/go-spew v1.1.3-alpha24 // indirect
	github.com/stackql/stackql-provider-registry v0.0.1-rc01 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/exp v0.0.0-20220722155223-a9213eeb770e // indirect
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3 // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/sys v0.0.0-20220310020820-b874c991c1a5 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.10 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.44.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/ini.v1 v1.66.2 // indirect
	gopkg.in/yaml.v3 v3.0.0 // indirect
)

replace github.com/chzyer/readline => github.com/stackql/readline v0.0.2-alpha05

replace github.com/fatih/color => github.com/stackql/color v0.0.1-rc01

replace vitess.io/vitess => github.com/stackql/vitess v0.0.11-alpha10

replace github.com/jeroenrinzema/psql-wire => github.com/stackql/psql-wire v0.0.2-stackqlbeta01
