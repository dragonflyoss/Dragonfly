/*
 *     Copyright 2020 The Dragonfly Authors
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
	"errors"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"d7y.io/dragonfly/v2/cmd/dependency/base"
	"d7y.io/dragonfly/v2/pkg/net/ip"
	"d7y.io/dragonfly/v2/pkg/types"
)

type Config struct {
	// Base options.
	base.Options `yaml:",inline" mapstructure:",squash"`

	// Server configuration.
	Server ServerConfig `yaml:"server" mapstructure:"server"`

	// Auth configuration.
	Auth AuthConfig `yaml:"auth" mapstructure:"auth"`

	// Database configuration.
	Database DatabaseConfig `yaml:"database" mapstructure:"database"`

	// Cache configuration.
	Cache CacheConfig `yaml:"cache" mapstructure:"cache"`

	// Job configuration.
	Job JobConfig `yaml:"job" mapstructure:"job"`

	// Metrics configuration.
	Metrics MetricsConfig `yaml:"metrics" mapstructure:"metrics"`

	// Network configuration.
	Network NetworkConfig `yaml:"network" mapstructure:"network"`
}

type ServerConfig struct {
	// Server name.
	Name string `yaml:"name" mapstructure:"name"`

	// Server log directory.
	LogDir string `yaml:"logDir" mapstructure:"logDir"`

	// LogLevel is log level of server, supported values are "debug", "info", "warn", "error", "panic", "fatal".
	LogLevel string `yaml:"logLevel" mapstructure:"logLevel"`

	// Maximum size in megabytes of log files before rotation (default: 1024)
	LogMaxSize int `yaml:"logMaxSize" mapstructure:"logMaxSize"`

	// Maximum number of days to retain old log files (default: 7)
	LogMaxAge int `yaml:"logMaxAge" mapstructure:"logMaxAge"`

	// Maximum number of old log files to keep (default: 20)
	LogMaxBackups int `yaml:"logMaxBackups" mapstructure:"logMaxBackups"`

	// Server plugin directory.
	PluginDir string `yaml:"pluginDir" mapstructure:"pluginDir"`

	// GRPC server configuration.
	GRPC GRPCConfig `yaml:"grpc" mapstructure:"grpc"`

	// REST server configuration.
	REST RESTConfig `yaml:"rest" mapstructure:"rest"`
}

type AuthConfig struct {
	// JWT configuration.
	JWT JWTConfig `yaml:"jwt" mapstructure:"jwt"`
}

type JWTConfig struct {
	// Realm name to display to the user, default value is Dragonfly.
	Realm string `yaml:"realm" mapstructure:"realm"`

	// Key is secret key used for signing. Please change the key in production
	Key string `yaml:"key" mapstructure:"key"`

	// Timeout is duration that a jwt token is valid, default duration is two days.
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`

	// MaxRefresh field allows clients to refresh their token until MaxRefresh has passed, default duration is two days.
	MaxRefresh time.Duration `yaml:"maxRefresh" mapstructure:"maxRefresh"`
}

type DatabaseConfig struct {
	// Database type.
	Type string `yaml:"type" mapstructure:"type"`

	// Mysql configuration.
	Mysql MysqlConfig `yaml:"mysql" mapstructure:"mysql"`

	// Postgres configuration.
	Postgres PostgresConfig `yaml:"postgres" mapstructure:"postgres"`

	// Polardb configuration.
	Polardb PolardbConfig `yaml:"polardb" mapstructure:"polardb"`

	// Redis configuration.
	Redis RedisConfig `yaml:"redis" mapstructure:"redis"`
}

type MysqlConfig struct {
	// Server username.
	User string `yaml:"user" mapstructure:"user"`

	// Server password.
	Password string `yaml:"password" mapstructure:"password"`

	// Server host.
	Host string `yaml:"host" mapstructure:"host"`

	// Server port.
	Port int `yaml:"port" mapstructure:"port"`

	// Server DB name.
	DBName string `yaml:"dbname" mapstructure:"dbname"`

	// TLS mode (can be one of "true", "false", "skip-verify",  or "preferred").
	TLSConfig string `yaml:"tlsConfig" mapstructure:"tlsConfig"`

	// Custom TLS client configuration (overrides "TLSConfig" setting above).
	TLS *MysqlTLSClientConfig `yaml:"tls" mapstructure:"tls"`

	// Enable migration.
	Migrate bool `yaml:"migrate" mapstructure:"migrate"`
}

type MysqlTLSClientConfig struct {
	// CACert is the file path of CA certificate for mysql.
	CACert string `yaml:"caCert" mapstructure:"caCert"`

	// Cert is the client certificate file path.
	Cert string `yaml:"cert" mapstructure:"cert"`

	// Key is the client key file path.
	Key string `yaml:"key" mapstructure:"key"`

	// InsecureSkipVerify controls whether a client verifies the
	// server's certificate chain and host name.
	InsecureSkipVerify bool `yaml:"insecureSkipVerify" mapstructure:"insecureSkipVerify"`
}

type PostgresConfig struct {
	// Server username.
	User string `yaml:"user" mapstructure:"user"`

	// Server password.
	Password string `yaml:"password" mapstructure:"password"`

	// Server host.
	Host string `yaml:"host" mapstructure:"host"`

	// Server port.
	Port int `yaml:"port" mapstructure:"port"`

	// Server DB name.
	DBName string `yaml:"dbname" mapstructure:"dbname"`

	// SSL mode.
	SSLMode string `yaml:"sslMode" mapstructure:"sslMode"`

	// Disable prepared statement.
	PreferSimpleProtocol bool `yaml:"preferSimpleProtocol" mapstructure:"preferSimpleProtocol"`

	// Server timezone.
	Timezone string `yaml:"timezone" mapstructure:"timezone"`

	// Enable migration.
	Migrate bool `yaml:"migrate" mapstructure:"migrate"`
}

type PolardbConfig struct {
	// Server username.
	User string `yaml:"user" mapstructure:"user"`

	// Server password.
	Password string `yaml:"password" mapstructure:"password"`

	// Server host:port list, format: "host1:port1,host2:port2".
	AddrList string `yaml:"addrList" mapstructure:"addrList"`

	// Server DB name.
	DBName string `yaml:"dbname" mapstructure:"dbname"`

	// Enable migration.
	Migrate bool `yaml:"migrate" mapstructure:"migrate"`
}

type RedisProxyConfig struct {
	// Enable redis proxy.
	Enable bool `yaml:"enable" mapstructure:"enable"`

	// Proxy address to listen on, default is ":65100".
	Addr string `yaml:"addr" mapstructure:"addr"`
}

// RedisConfig is redis configuration.
// see: https://redis.uptrace.dev/guide/universal.html
type RedisConfig struct {
	// DEPRECATED: Please use the `addrs` field instead.
	Host string `yaml:"host" mapstructure:"host"`

	// DEPRECATED: Please use the `addrs` field instead.
	Port int `yaml:"port" mapstructure:"port"`

	// Addrs is server addresses.
	Addrs []string `yaml:"addrs" mapstructure:"addrs"`

	// MasterName is the sentinel master name.
	MasterName string `yaml:"masterName" mapstructure:"masterName"`

	// Username is server username.
	Username string `yaml:"username" mapstructure:"username"`

	// Password is server password.
	Password string `yaml:"password" mapstructure:"password"`

	// SentinelUsername is sentinel server username.
	SentinelUsername string `yaml:"sentinelUsername" mapstructure:"sentinelUsername"`

	// SentinelPassword is sentinel server password.
	SentinelPassword string `yaml:"sentinelPassword" mapstructure:"sentinelPassword"`

	// DB is server cache DB name.
	DB int `yaml:"db" mapstructure:"db"`

	// BrokerDB is server broker DB name.
	BrokerDB int `yaml:"brokerDB" mapstructure:"brokerDB"`

	// BackendDB is server backend DB name.
	BackendDB int `yaml:"backendDB" mapstructure:"backendDB"`

	// PoolSize is the maximum number of connections in the idle connection pool.
	PoolSize int `yaml:"poolSize" mapstructure:"poolSize"`

	// PoolTimeout is the maximum amount of time a connection may be idle before being closed.
	PoolTimeout time.Duration `yaml:"poolTimeout" mapstructure:"poolTimeout"`

	// Proxy is redis proxy configuration.
	// If enabled, the manager starts a TCP proxy (defaulting to port 65100) that
	// forwards requests to the Redis service. This allows Schedulers to connect to Redis
	// via the manager's local port, which is useful for network isolation as only the
	// manager's IP and port need to be exposed.
	// Note: Only a single Redis address is supported by the proxy.
	Proxy RedisProxyConfig `yaml:"proxy" mapstructure:"proxy"`

	// Custom TLS client configuration for connecting to redis.
	TLS *RedisTLSClientConfig `yaml:"tls" mapstructure:"tls"`
}

type RedisTLSClientConfig struct {
	// CACert is the file path of CA certificate for redis.
	CACert string `yaml:"caCert" mapstructure:"caCert"`

	// Cert is the client certificate file path.
	Cert string `yaml:"cert" mapstructure:"cert"`

	// Key is the client key file path.
	Key string `yaml:"key" mapstructure:"key"`

	// InsecureSkipVerify controls whether a client verifies the
	// server's certificate chain and host name.
	InsecureSkipVerify bool `yaml:"insecureSkipVerify" mapstructure:"insecureSkipVerify"`
}

type CacheConfig struct {
	// Redis cache configuration.
	Redis RedisCacheConfig `yaml:"redis" mapstructure:"redis"`

	// Local cache configuration.
	Local LocalCacheConfig `yaml:"local" mapstructure:"local"`
}

type RedisCacheConfig struct {
	// Cache TTL.
	TTL time.Duration `yaml:"ttl" mapstructure:"ttl"`
}

type LocalCacheConfig struct {
	// Size of LFU cache.
	Size int `yaml:"size" mapstructure:"size"`

	// Cache TTL.
	TTL time.Duration `yaml:"ttl" mapstructure:"ttl"`
}

type RESTConfig struct {
	// REST server address.
	Addr string `yaml:"addr" mapstructure:"addr"`

	// TLS server configuration.
	TLS *RESTTLSServerConfig `yaml:"tls" mapstructure:"tls"`
}

type RESTTLSServerConfig struct {
	// Certificate file path.
	Cert string `yaml:"cert" mapstructure:"cert"`

	// Key file path.
	Key string `yaml:"key" mapstructure:"key"`
}

type MetricsConfig struct {
	// Enable metrics service.
	Enable bool `yaml:"enable" mapstructure:"enable"`

	// Metrics service address.
	Addr string `yaml:"addr" mapstructure:"addr"`
}

// GRPCConfig is gRPC server configuration.
type GRPCConfig struct {
	// AdvertiseIP is advertise ip.
	AdvertiseIP net.IP `yaml:"advertiseIP" mapstructure:"advertiseIP"`

	// ListenIP is listen ip, like: 0.0.0.0, 192.168.0.1.
	ListenIP net.IP `mapstructure:"listenIP" yaml:"listenIP"`

	// Port is listen port.
	Port TCPListenPortRange `yaml:"port" mapstructure:"port"`

	// TLS server configuration.
	TLS *GRPCTLSServerConfig `yaml:"tls" mapstructure:"tls"`

	// RequestRateLimit is the maximum number of requests per second for the gRPC server.
	// It limits both the rate of unary gRPC requests and the rate of new stream gRPC connection.
	RequestRateLimit float64 `yaml:"requestRateLimit" mapstructure:"requestRateLimit"`
}

type TCPListenPortRange struct {
	// Start is the start port.
	Start int

	// End is the end port.
	End int
}

type GRPCTLSServerConfig struct {
	// CACert is the file path of CA certificate for mTLS.
	CACert string `yaml:"caCert" mapstructure:"caCert"`

	// Cert is the file path of server certificate for mTLS.
	Cert string `yaml:"cert" mapstructure:"cert"`

	// Key is the file path of server key for mTLS.
	Key string `yaml:"key" mapstructure:"key"`
}

type JobConfig struct {
	// Preheat configuration.
	Preheat PreheatConfig `yaml:"preheat" mapstructure:"preheat"`

	// Sync peers configuration.
	SyncPeers SyncPeersConfig `yaml:"syncPeers" mapstructure:"syncPeers"`
}

type PreheatConfig struct {
	// TLS client configuration.
	TLS PreheatTLSClientConfig `yaml:"tls" mapstructure:"tls"`
}

type SyncPeersConfig struct {
	// Interval is the interval for syncing all peers information from the scheduler and
	// display peers information in the manager console.
	Interval time.Duration `yaml:"interval" mapstructure:"interval"`

	// Timeout is the timeout for syncing peers information from the single scheduler.
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`

	// BatchSize is the batch size when operating gorm database.
	BatchSize int `yaml:"batchSize" mapstructure:"batchSize"`

	// SyncBatchSize is the batch size when get host info from scheduler.
	SyncBatchSize int `yaml:"syncBatchSize" mapstructure:"syncBatchSize"`
}

type PreheatTLSClientConfig struct {
	// InsecureSkipVerify controls whether a client verifies the
	// server's certificate chain and host name.
	InsecureSkipVerify bool `yaml:"insecureSkipVerify" mapstructure:"insecureSkipVerify"`

	// CACert is the file path of CA certificate for preheating.
	CACert types.PEMContent `yaml:"caCert" mapstructure:"caCert"`
}

type NetworkConfig struct {
	// EnableIPv6 enables ipv6 for server.
	EnableIPv6 bool `mapstructure:"enableIPv6" yaml:"enableIPv6"`
}

// New config instance.
func New() *Config {
	return &Config{
		Options: base.Options{
			Console:   false,
			PProfPort: -1,
			Tracing: base.TracingConfig{
				Path:        "/v1/traces",
				ServiceName: types.ManagerName,
			},
		},
		Server: ServerConfig{
			Name: DefaultServerName,
			GRPC: GRPCConfig{
				Port: TCPListenPortRange{
					Start: DefaultGRPCPort,
					End:   DefaultGRPCPort,
				},
				RequestRateLimit: DefaultServerGRPCRequestRateLimit,
			},
			REST: RESTConfig{
				Addr: DefaultRESTAddr,
			},
			LogLevel:      "info",
			LogMaxSize:    DefaultLogRotateMaxSize,
			LogMaxAge:     DefaultLogRotateMaxAge,
			LogMaxBackups: DefaultLogRotateMaxBackups,
		},
		Auth: AuthConfig{
			JWT: JWTConfig{
				Realm:      DefaultJWTRealm,
				Timeout:    DefaultJWTTimeout,
				MaxRefresh: DefaultJWTMaxRefresh,
			},
		},
		Database: DatabaseConfig{
			Type: DatabaseTypeMysql,
			Mysql: MysqlConfig{
				Port:    DefaultMysqlPort,
				DBName:  DefaultMysqlDBName,
				Migrate: true,
			},
			Postgres: PostgresConfig{
				Port:                 DefaultPostgresPort,
				DBName:               DefaultPostgresDBName,
				SSLMode:              DefaultPostgresSSLMode,
				PreferSimpleProtocol: DefaultPostgresPreferSimpleProtocol,
				Timezone:             DefaultPostgresTimezone,
				Migrate:              true,
			},
			Polardb: PolardbConfig{
				DBName:  DefaultPolardbDBName,
				Migrate: true,
			},
			Redis: RedisConfig{
				DB:          DefaultRedisDB,
				BrokerDB:    DefaultRedisBrokerDB,
				BackendDB:   DefaultRedisBackendDB,
				PoolSize:    DefaultRedisPoolSize,
				PoolTimeout: DefaultRedisPoolTimeout,
				Proxy: RedisProxyConfig{
					Enable: false,
					Addr:   DefaultRedisProxyAddr,
				},
			},
		},
		Cache: CacheConfig{
			Redis: RedisCacheConfig{
				TTL: DefaultRedisCacheTTL,
			},
			Local: LocalCacheConfig{
				Size: DefaultLFUCacheSize,
				TTL:  DefaultLFUCacheTTL,
			},
		},
		Job: JobConfig{
			Preheat: PreheatConfig{
				TLS: PreheatTLSClientConfig{},
			},
			SyncPeers: SyncPeersConfig{
				Interval:      DefaultJobSyncPeersInterval,
				Timeout:       DefaultJobSyncPeersTimeout,
				BatchSize:     DefaultJobSyncPeersDatabaseBatchSize,
				SyncBatchSize: DefaultJobSyncPeersBatchSize,
			},
		},
		Metrics: MetricsConfig{
			Enable: false,
			Addr:   DefaultMetricsAddr,
		},
		Network: NetworkConfig{
			EnableIPv6: DefaultNetworkEnableIPv6,
		},
	}
}

// Validate config values
func (cfg *Config) Validate() error {
	if cfg.Server.Name == "" {
		return errors.New("server requires parameter name")
	}

	if cfg.Server.GRPC.AdvertiseIP == nil {
		return errors.New("grpc requires parameter advertiseIP")
	}

	if cfg.Server.GRPC.ListenIP == nil {
		return errors.New("grpc requires parameter listenIP")
	}

	if cfg.Server.GRPC.TLS != nil {
		if cfg.Server.GRPC.TLS.CACert == "" {
			return errors.New("grpc tls requires parameter caCert")
		}

		if cfg.Server.GRPC.TLS.Cert == "" {
			return errors.New("grpc tls requires parameter cert")
		}

		if cfg.Server.GRPC.TLS.Key == "" {
			return errors.New("grpc tls requires parameter key")
		}
	}

	if cfg.Server.GRPC.RequestRateLimit <= 0 {
		return errors.New("grpc requires parameter requestRateLimit")
	}

	if cfg.Server.REST.TLS != nil {
		if cfg.Server.REST.TLS.Cert == "" {
			return errors.New("rest tls requires parameter cert")
		}

		if cfg.Server.REST.TLS.Key == "" {
			return errors.New("rest tls requires parameter key")
		}
	}

	if cfg.Auth.JWT.Realm == "" {
		return errors.New("jwt requires parameter realm")
	}

	if cfg.Auth.JWT.Key == "" {
		return errors.New("jwt requires parameter key")
	}

	if cfg.Auth.JWT.Timeout == 0 {
		return errors.New("jwt requires parameter timeout")
	}

	if cfg.Auth.JWT.MaxRefresh == 0 {
		return errors.New("jwt requires parameter maxRefresh")
	}

	if cfg.Database.Type == "" {
		return errors.New("database requires parameter type")
	}

	if slices.Contains([]string{DatabaseTypeMysql, DatabaseTypeMariaDB}, cfg.Database.Type) {
		if cfg.Database.Mysql.User == "" {
			return errors.New("mysql requires parameter user")
		}

		if cfg.Database.Mysql.Password == "" {
			return errors.New("mysql requires parameter password")
		}

		if cfg.Database.Mysql.Host == "" {
			return errors.New("mysql requires parameter host")
		}

		if cfg.Database.Mysql.Port <= 0 {
			return errors.New("mysql requires parameter port")
		}

		if cfg.Database.Mysql.DBName == "" {
			return errors.New("mysql requires parameter dbname")
		}

		if cfg.Database.Mysql.TLS != nil {
			if cfg.Database.Mysql.TLS.CACert == "" {
				return errors.New("mysql tls requires parameter caCert")
			}

			if cfg.Database.Mysql.TLS.Cert == "" {
				return errors.New("mysql tls requires parameter cert")
			}

			if cfg.Database.Mysql.TLS.Key == "" {
				return errors.New("mysql tls requires parameter key")
			}
		}
	}

	if cfg.Database.Type == DatabaseTypePostgres {
		if cfg.Database.Postgres.User == "" {
			return errors.New("postgres requires parameter user")
		}

		if cfg.Database.Postgres.Password == "" {
			return errors.New("postgres requires parameter password")
		}

		if cfg.Database.Postgres.Host == "" {
			return errors.New("postgres requires parameter host")
		}

		if cfg.Database.Postgres.Port <= 0 {
			return errors.New("postgres requires parameter port")
		}

		if cfg.Database.Postgres.DBName == "" {
			return errors.New("postgres requires parameter dbname")
		}

		if cfg.Database.Postgres.SSLMode == "" {
			return errors.New("postgres requires parameter sslMode")
		}

		if cfg.Database.Postgres.Timezone == "" {
			return errors.New("postgres requires parameter timezone")
		}
	}

	if cfg.Database.Type == DatabaseTypePolardb {
		if cfg.Database.Polardb.User == "" {
			return errors.New("polardb requires parameter user")
		}

		if cfg.Database.Polardb.Password == "" {
			return errors.New("polardb requires parameter password")
		}

		if cfg.Database.Polardb.AddrList == "" || !strings.Contains(cfg.Database.Polardb.AddrList, ":") {
			return errors.New("polardb requires parameter addrList, format: \"host1:port1,host2:port2\"")
		}

		if cfg.Database.Polardb.DBName == "" {
			return errors.New("polardb requires parameter dbname")
		}
	}

	if len(cfg.Database.Redis.Addrs) == 0 {
		return errors.New("redis requires parameter addrs")
	}

	if cfg.Database.Redis.DB < 0 {
		return errors.New("redis requires parameter db")
	}

	if cfg.Database.Redis.BrokerDB < 0 {
		return errors.New("redis requires parameter brokerDB")
	}

	if cfg.Database.Redis.BackendDB < 0 {
		return errors.New("redis requires parameter backendDB")
	}

	if cfg.Database.Redis.Proxy.Enable {
		if cfg.Database.Redis.Proxy.Addr == "" {
			return errors.New("redis proxy requires parameter addr")
		}
	}

	if cfg.Database.Redis.TLS != nil {
		tls := cfg.Database.Redis.TLS
		if !tls.InsecureSkipVerify {
			if tls.CACert == "" {
				return errors.New("redis tls requires parameter caCert or insecureSkipVerify")
			}

			if (tls.Cert == "") != (tls.Key == "") {
				return errors.New("redis tls cert and key must be provided together")
			}
		}
	}

	if cfg.Cache.Redis.TTL == 0 {
		return errors.New("redis requires parameter ttl")
	}

	if cfg.Cache.Local.Size == 0 {
		return errors.New("local requires parameter size")
	}

	if cfg.Cache.Local.TTL == 0 {
		return errors.New("local requires parameter ttl")
	}

	if cfg.Job.SyncPeers.Interval <= MinJobSyncPeersInterval {
		return errors.New("syncPeers requires parameter interval and it must be greater than 12 hours")
	}

	if cfg.Job.SyncPeers.Timeout == 0 {
		return errors.New("syncPeers requires parameter timeout")
	}

	if cfg.Job.SyncPeers.BatchSize == 0 {
		return errors.New("syncPeers requires parameter batchSize")
	}

	if cfg.Metrics.Enable {
		if cfg.Metrics.Addr == "" {
			return errors.New("metrics requires parameter addr")
		}
	}

	return nil
}

func (cfg *Config) Convert() error {
	// TODO Compatible with deprecated fields host and port.
	if len(cfg.Database.Redis.Addrs) == 0 && cfg.Database.Redis.Host != "" && cfg.Database.Redis.Port > 0 {
		cfg.Database.Redis.Addrs = []string{net.JoinHostPort(cfg.Database.Redis.Host, strconv.Itoa(cfg.Database.Redis.Port))}
	}

	if cfg.Server.GRPC.AdvertiseIP == nil {
		if cfg.Network.EnableIPv6 {
			cfg.Server.GRPC.AdvertiseIP = ip.IPv6
		} else {
			cfg.Server.GRPC.AdvertiseIP = ip.IPv4
		}
	}

	if cfg.Server.GRPC.ListenIP == nil {
		if cfg.Network.EnableIPv6 {
			cfg.Server.GRPC.ListenIP = net.IPv6zero
		} else {
			cfg.Server.GRPC.ListenIP = net.IPv4zero
		}
	}

	return nil
}
