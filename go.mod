module github.com/stackql/stackql

go 1.16

require (
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40 // indirect
	github.com/axiomhq/hyperloglog v0.0.0-20220105174342-98591331716a // indirect
	github.com/biogo/store v0.0.0-20201120204734-aad293a2328f // indirect
	github.com/cenk/backoff v2.2.1+incompatible // indirect
	github.com/certifi/gocertifi v0.0.0-20210507211836-431795d63e8d // indirect
	github.com/cockroachdb/circuitbreaker v2.2.1+incompatible // indirect
	github.com/cockroachdb/cmux v0.0.0-20170110192607-30d10be49292 // indirect
	github.com/cockroachdb/cockroach v20.1.17+incompatible
	github.com/cockroachdb/cockroach-go v2.0.1+incompatible // indirect
	github.com/cockroachdb/errors v1.9.0 // indirect
	github.com/cockroachdb/ttycolor v0.0.0-20210902133924-c7d7dcdde4e8 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/color v1.13.0
	github.com/getkin/kin-openapi v0.88.0
	github.com/getsentry/raven-go v0.2.0 // indirect
	github.com/google/flatbuffers v2.0.6+incompatible // indirect
	github.com/google/go-jsonnet v0.17.0
	github.com/jackc/pgtype v1.10.0
	github.com/jackc/pgx v3.6.2+incompatible // indirect
	github.com/jaegertracing/jaeger v1.32.0 // indirect
	github.com/jeroenrinzema/psql-wire v0.0.1-stackqlalpha3
	github.com/knz/strtime v0.0.0-20200924090105-187c67f2bf5e // indirect
	github.com/lib/pq v1.10.4
	github.com/lightstep/lightstep-tracer-go v0.25.0 // indirect
	github.com/magiconair/properties v1.8.6
	github.com/marusama/semaphore v0.0.0-20190110074507-6952cef993b2 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/montanaflynn/stats v0.6.6 // indirect
	github.com/olekukonko/tablewriter v0.0.0-20180130162743-b8a9be070da4
	github.com/openzipkin-contrib/zipkin-go-opentracing v0.4.5 // indirect
	github.com/petermattis/goid v0.0.0-20220302125637-5f11c28912df // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stackql/go-openapistackql v0.0.6-beta01
	github.com/stackql/go-sqlite3 v0.0.1-stackqlrc01
	github.com/stackql/go-suffix-map v0.0.1-alpha01
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	gonum.org/v1/gonum v0.9.3
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	readline v0.0.0-00010101000000-000000000000
	vitess.io/vitess v0.0.11-alpha04
)

replace readline => github.com/stackql/readline v0.0.1-rc01

replace github.com/fatih/color => github.com/stackql/color v0.0.1-rc01

replace vitess.io/vitess => github.com/stackql/vitess v0.0.11-alpha04

replace github.com/jeroenrinzema/psql-wire => github.com/stackql/psql-wire v0.0.1-stackqlrc01
