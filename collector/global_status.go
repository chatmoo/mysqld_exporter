// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Scrape `SHOW GLOBAL STATUS`.

package collector

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Scrape query.
	globalStatusQuery = `SHOW GLOBAL STATUS`
	// Subsystem.
	globalStatus = "global_status"
)

// Regexp to match various groups of status vars.
var globalStatusRE = regexp.MustCompile(`^(com|handler|connection_errors|innodb_buffer_pool_pages|innodb_system_rows|innodb_sampled|performance_schema|current_tls|ssl|mysqlx|binlog_stmt_cache)_(.*)$`)

// Metric descriptors.
var (
	globalCommandsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "commands_total"),
		"Total number of executed MySQL commands.",
		[]string{"command"}, nil,
	)
	globalHandlerDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "handlers_total"),
		"Total number of executed MySQL handlers.",
		[]string{"handler"}, nil,
	)
	globalConnectionErrorsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "connection_errors_total"),
		"Total number of MySQL connection errors.",
		[]string{"error"}, nil,
	)
	globalBufferPoolPagesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "buffer_pool_pages"),
		"Innodb buffer pool pages by state.",
		[]string{"state"}, nil,
	)
	globalBufferPoolDirtyPagesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "buffer_pool_dirty_pages"),
		"Innodb buffer pool dirty pages.",
		[]string{}, nil,
	)
	globalBufferPoolPageChangesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "buffer_pool_page_changes_total"),
		"Innodb buffer pool page state changes.",
		[]string{"operation"}, nil,
	)
	globalInnoDBRowOpsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, globalStatus, "innodb_row_ops_total"),
		"Total number of MySQL InnoDB row operations.",
		[]string{"operation"}, nil,
	)
)

// ScrapeGlobalStatus collects from `SHOW GLOBAL STATUS`.
type ScrapeGlobalStatus struct{}

// Name of the Scraper. Should be unique.
func (ScrapeGlobalStatus) Name() string {
	return globalStatus
}

// Help describes the role of the Scraper.
func (ScrapeGlobalStatus) Help() string {
	return "Collect from SHOW GLOBAL STATUS"
}

// Version of MySQL from which scraper is available.
func (ScrapeGlobalStatus) Version() float64 {
	return 5.1
}

// Scrape collects data from database connection and sends it over channel as prometheus metric.
func (ScrapeGlobalStatus) Scrape(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric, logger log.Logger) error {
	globalStatusRows, err := db.QueryContext(ctx, globalStatusQuery)
	if err != nil {
		return err
	}
	defer globalStatusRows.Close()

	var key string
	var val sql.RawBytes

	for globalStatusRows.Next() {
		if err := globalStatusRows.Scan(&key, &val); err != nil {
			return err
		}
		if floatVal, ok := parseStatus(val); ok { // Unparsable values are silently skipped.
			key = validPrometheusName(key)
			match := globalStatusRE.FindStringSubmatch(key)
			if match == nil {
				ch <- prometheus.MustNewConstMetric(
					newDesc(globalStatus, key, "Generic metric from SHOW GLOBAL STATUS."),
					prometheus.UntypedValue,
					floatVal,
				)
				continue
			}
			switch match[1] {
			case "com":
				switch match[2] {
				case "begin", "commit", "rollback", "create_trigger", "create_view", "group_replication_start", "group_replication_stop":
					ch <- prometheus.MustNewConstMetric(
						globalCommandsDesc, prometheus.CounterValue, floatVal, match[2],
					)
				default:
					continue
				}
			case "handler":
				ch <- prometheus.MustNewConstMetric(
					globalHandlerDesc, prometheus.CounterValue, floatVal, match[2],
				)
			case "connection_errors":
				ch <- prometheus.MustNewConstMetric(
					globalConnectionErrorsDesc, prometheus.CounterValue, floatVal, match[2],
				)
			case "innodb_buffer_pool_pages":
				switch match[2] {
				case "data", "free", "misc", "old":
					ch <- prometheus.MustNewConstMetric(
						globalBufferPoolPagesDesc, prometheus.GaugeValue, floatVal, match[2],
					)
				case "dirty":
					ch <- prometheus.MustNewConstMetric(
						globalBufferPoolDirtyPagesDesc, prometheus.GaugeValue, floatVal,
					)
				case "total":
					continue
				default:
					ch <- prometheus.MustNewConstMetric(
						globalBufferPoolPageChangesDesc, prometheus.CounterValue, floatVal, match[2],
					)
				}
			case "innodb_rows":
				ch <- prometheus.MustNewConstMetric(
					globalInnoDBRowOpsDesc, prometheus.CounterValue, floatVal, match[2],
				)
			case "ssl":
				continue
			case "mysqlx":
				continue
			case "performance_schema":
				continue
			case "innodb_sampled":
				continue
			case "innodb_system_rows":
				continue
			case "binlog_stmt_cache":
				continue
			}
		}
	}
	return nil
}

func validPrometheusName(s string) string {
	nameRe := regexp.MustCompile("([^a-zA-Z0-9_])")
	s = nameRe.ReplaceAllString(s, "_")
	s = strings.ToLower(s)
	return s
}

// check interface
var _ Scraper = ScrapeGlobalStatus{}
