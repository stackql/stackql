module github.com/stackql/stackql

go 1.16

require (
	github.com/bketelsen/crypt v0.0.4 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.13.0
	github.com/getkin/kin-openapi v0.88.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-delve/delve v1.6.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.5
	github.com/google/go-jsonnet v0.17.0
	github.com/magiconair/properties v1.8.5
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/olekukonko/tablewriter v0.0.0-20180130162743-b8a9be070da4
	github.com/peterh/liner v1.2.1 // indirect
	github.com/pkg/profile v0.0.0-20170413231811-06b906832ed0 // indirect
	github.com/prometheus/common v0.9.1
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stackql/go-openapistackql v0.0.4-alpha46
	github.com/stackql/go-sqlite3 v0.0.1-stackqlalpha
	go.starlark.net v0.0.0-20210602144842-1cdb82c9e17a // indirect
	golang.org/x/arch v0.0.0-20210502124803-cbf565b21d1e // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sys v0.0.0-20211210111614-af8b64212486
	gonum.org/v1/gonum v0.0.0-20190331200053-3d26580ed485
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0
	readline v0.0.0-00010101000000-000000000000
	vitess.io/vitess v0.0.9-alpha7
)

replace readline => github.com/stackql/readline v0.0.0-20210418072316-6e4ad520d2b4

replace github.com/fatih/color => github.com/stackql/color v1.10.1-0.20210418074258-4aa529ee76ed

replace vitess.io/vitess => github.com/stackql/vitess v0.0.9-alpha7
