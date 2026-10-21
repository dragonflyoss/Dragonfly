/*
 *     Copyright 2022 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"net"
	"time"

	"d7y.io/dragonfly/v2/pkg/net/ip"
)

const (
	// DatabaseTypeMysql is database type of mysql.
	DatabaseTypeMysql = "mysql"

	// DatabaseTypeMariaDB is database type of mariadb.
	DatabaseTypeMariaDB = "mariadb"

	// DatabaseTypePostgres is database type of postgres.
	DatabaseTypePostgres = "postgres"

	// DatabaseTypePolardb is database type of polardb.
	DatabaseTypePolardb = "polardb"
)

const (
	// DefaultServerName is default server name.
	DefaultServerName = "d7y/manager"

	// DefaultGRPCPort is default port for grpc server.
	DefaultGRPCPort = 65003

	// DefaultServerGRPCRequestRateLimit is the default number of requests per second for the gRPC server.
	// It limits both the rate of unary gRPC requests and the rate of new stream gRPC connection.
	DefaultServerGRPCRequestRateLimit = 4000

	// DefaultRESTAddr is default address for rest server.
	DefaultRESTAddr = ":8080"
)

const (
	// DefaultJWTTimeout is default of name in jwt.
	DefaultJWTRealm = "Dragonfly"

	// DefaultJWTTimeout is default of timeout in jwt.
	DefaultJWTTimeout = 14 * 24 * time.Hour

	// DefaultJWTMaxRefresh is default of max refresh in jwt.
	DefaultJWTMaxRefresh = 7 * 24 * time.Hour
)

const (
	// DefaultRedisPoolSize is default pool size for redis.
	DefaultRedisPoolSize = 40

	// DefaultRedisPoolTimeout is default pool timeout for redis.
	DefaultRedisPoolTimeout = 10 * time.Second

	// DefaultRedisDB is default db for redis.
	DefaultRedisDB = 0

	// DefaultRedisBrokerDB is default db for redis broker.
	DefaultRedisBrokerDB = 1

	// DefaultRedisBackendDB is default db for redis backend.
	DefaultRedisBackendDB = 2

	// DefaultRedisProxyAddr is default address for redis proxy.
	DefaultRedisProxyAddr = ":65100"
)

const (
	// DefaultRedisCacheTTL is default ttl for redis cache.
	DefaultRedisCacheTTL = 5 * time.Minute

	// DefaultLFUCacheTTL is default ttl for lfu cache.
	DefaultLFUCacheTTL = 3 * time.Minute

	// DefaultLFUCacheSize is default size for lfu cache.
	DefaultLFUCacheSize = 8 * 1000
)

const (
	// DefaultMysqlPort is default port for mysql.
	DefaultMysqlPort = 3306

	// DefaultMysqlDBName is default db name for mysql.
	DefaultMysqlDBName = "manager"
)

const (
	// DefaultJobSyncPeersInterval is the default interval for syncing all peers information from the scheduler.
	DefaultJobSyncPeersInterval = 24 * time.Hour

	// MinJobSyncPeersInterval is the min interval for syncing all peers information from the scheduler.
	MinJobSyncPeersInterval = 12 * time.Hour

	// DefaultJobSyncPeersTimeout is the default timeout for syncing all peers information from the scheduler.
	DefaultJobSyncPeersTimeout = 10 * time.Minute

	// DefaultClusterJobRateLimit is default rate limit(requests per second) for job Open API by cluster.
	DefaultClusterJobRateLimit uint32 = 10

	// DefaultJobSyncPeersDatabaseBatchSize is the default batch size for operating on the database.
	DefaultJobSyncPeersDatabaseBatchSize = 1

	// DefaultJobSyncPeersBatchSize is the default batch size for syncing all peers information from the scheduler.
	DefaultJobSyncPeersBatchSize = 500
)

const (
	// DefaultPostgresPort is default port for postgres.
	DefaultPostgresPort = 5432

	// DefaultPostgresDBName is default db name for postgres.
	DefaultPostgresDBName = "manager"

	// DefaultPostgresSSLMode is default ssl mode for postgres.
	DefaultPostgresSSLMode = "disable"

	// DefaultPostgresPreferSimpleProtocol is default disable prepared statement option for postgres.
	DefaultPostgresPreferSimpleProtocol = false

	// DefaultPostgresTimezone is default timezone for postgres.
	DefaultPostgresTimezone = "UTC"
)

const (
	// DefaultPolardbDBName is default db name for polardb.
	DefaultPolardbDBName = "manager"
)

const (
	// DefaultMetricsAddr is default address for metrics server.
	DefaultMetricsAddr = ":8000"
)

const (
	// DefaultLogRotateMaxSize is the default maximum size in megabytes of log files before rotation.
	DefaultLogRotateMaxSize = 1024

	// DefaultLogRotateMaxAge is the default number of days to retain old log files.
	DefaultLogRotateMaxAge = 7

	// DefaultLogRotateMaxBackups is the default number of old log files to keep.
	DefaultLogRotateMaxBackups = 20
)

var (
	// DefaultCertIPAddresses is default ip addresses of certificate.
	DefaultCertIPAddresses = []net.IP{ip.IPv4, ip.IPv6}

	// DefaultCertDNSNames is default dns names of certificate.
	DefaultCertDNSNames = []string{"dragonfly-manager", "dragonfly-manager.dragonfly-system.svc", "dragonfly-manager.dragonfly-system.svc.cluster.local"}

	// DefaultCertValidityPeriod is default validity period of certificate.
	DefaultCertValidityPeriod = 10 * 365 * 24 * time.Hour
)

var (
	// DefaultNetworkEnableIPv6 is default value of enableIPv6.
	DefaultNetworkEnableIPv6 = false
)
