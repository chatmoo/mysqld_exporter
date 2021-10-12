module mysqld_exporter

go 1.16

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/go-kit/log v0.2.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.31.1
	github.com/prometheus/exporter-toolkit v0.6.1
	github.com/smartystreets/goconvey v1.6.6
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/ini.v1 v1.63.2
)
