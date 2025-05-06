// services/backlog-service/internal/config/config.go

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Environment string        `mapstructure:"environment"`
	Server      ServerConfig  `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Cache       CacheConfig   `mapstructure:"cache"`
	EventBus    KafkaConfig   `mapstructure:"event_bus"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Security    SecurityConfig `mapstructure:"security"`
}

// ServerConfig holds configuration for the server
type ServerConfig struct {
	GRPCPort     int           `mapstructure:"grpc_port"`
	HTTPPort     int           `mapstructure:"http_port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	GracefulShutdownTimeout time.Duration `mapstructure:"graceful_shutdown_timeout"`
}

// DatabaseConfig holds configuration for the database
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// CacheConfig holds configuration for the cache
type CacheConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	TLSEnabled   bool          `mapstructure:"tls_enabled"`
}

// KafkaConfig holds configuration for Kafka
type KafkaConfig struct {
	BootstrapServers string `mapstructure:"bootstrap_servers"`
	ClientID         string `mapstructure:"client_id"`
	SecurityProtocol string `mapstructure:"security_protocol"`
	SASLMechanism    string `mapstructure:"sasl_mechanism"`
	SASLUsername     string `mapstructure:"sasl_username"`
	SASLPassword     string `mapstructure:"sasl_password"`
}

// ObservabilityConfig holds configuration for observability
type ObservabilityConfig struct {
	LogLevel            string `mapstructure:"log_level"`
	EnableStructuredLogs bool   `mapstructure:"enable_structured_logs"`
	TracingEnabled      bool   `mapstructure:"tracing_enabled"`
	TracingEndpoint     string `mapstructure:"tracing_endpoint"`
	MetricsEnabled      bool   `mapstructure:"metrics_enabled"`
	MetricsEndpoint     string `mapstructure:"metrics_endpoint"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	JWTSecret            string        `mapstructure:"jwt_secret"`
	JWTExpirationTime    time.Duration `mapstructure:"jwt_expiration_time"`
	AllowedOrigins       []string      `mapstructure:"allowed_origins"`
	EnableTLS            bool          `mapstructure:"enable_tls"`
	TLSCertFile          string        `mapstructure:"tls_cert_file"`
	TLSKeyFile           string        `mapstructure:"tls_key_file"`
	EnableRateLimiting   bool          `mapstructure:"enable_rate_limiting"`
	RateLimitPerSecond   int           `mapstructure:"rate_limit_per_second"`
	EnableRequestLogging bool          `mapstructure:"enable_request_logging"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set defaults
	setDefaultConfig()

	// Load .env file if exists
	_ = godotenv.Load()

	// Load configuration file
	configFile := getConfigFile()
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Unmarshal into config
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with explicit environment variables
	overrideWithEnvVars(&config)

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// setDefaultConfig sets default configuration values
func setDefaultConfig() {
	// Server defaults
	viper.SetDefault("server.grpc_port", 8080)
	viper.SetDefault("server.http_port", 8081)
	viper.SetDefault("server.read_timeout", 5*time.Second)
	viper.SetDefault("server.write_timeout", 10*time.Second)
	viper.SetDefault("server.graceful_shutdown_timeout", 30*time.Second)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.database", "ubmm")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// Cache defaults
	viper.SetDefault("cache.host", "localhost")
	viper.SetDefault("cache.port", 6379)
	viper.SetDefault("cache.password", "")
	viper.SetDefault("cache.db", 0)
	viper.SetDefault("cache.pool_size", 10)
	viper.SetDefault("cache.min_idle_conns", 2)
	viper.SetDefault("cache.dial_timeout", 5*time.Second)
	viper.SetDefault("cache.read_timeout", 3*time.Second)
	viper.SetDefault("cache.write_timeout", 3*time.Second)
	viper.SetDefault("cache.tls_enabled", false)

	// Kafka defaults
	viper.SetDefault("event_bus.bootstrap_servers", "localhost:9092")
	viper.SetDefault("event_bus.client_id", "backlog-service")
	viper.SetDefault("event_bus.security_protocol", "plaintext")
	viper.SetDefault("event_bus.sasl_mechanism", "")
	viper.SetDefault("event_bus.sasl_username", "")
	viper.SetDefault("event_bus.sasl_password", "")

	// Observability defaults
	viper.SetDefault("observability.log_level", "info")
	viper.SetDefault("observability.enable_structured_logs", true)
	viper.SetDefault("observability.tracing_enabled", true)
	viper.SetDefault("observability.tracing_endpoint", "localhost:4317")
	viper.SetDefault("observability.metrics_enabled", true)
	viper.SetDefault("observability.metrics_endpoint", "localhost:9090")

	// Security defaults
	viper.SetDefault("security.jwt_secret", "")
	viper.SetDefault("security.jwt_expiration_time", 24*time.Hour)
	viper.SetDefault("security.allowed_origins", []string{"*"})
	viper.SetDefault("security.enable_tls", false)
	viper.SetDefault("security.tls_cert_file", "")
	viper.SetDefault("security.tls_key_file", "")
	viper.SetDefault("security.enable_rate_limiting", true)
	viper.SetDefault("security.rate_limit_per_second", 100)
	viper.SetDefault("security.enable_request_logging", true)

	// Environment default
	viper.SetDefault("environment", "development")
}

// getConfigFile determines the configuration file path
func getConfigFile() string {
	// Check if config file is explicitly set
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		return configFile
	}

	// Check if config directory is explicitly set
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		configDir = "config"
	}

	// Determine environment for config
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	// Check for config file in order of precedence
	candidates := []string{
		fmt.Sprintf("%s/%s.yaml", configDir, env),
		fmt.Sprintf("%s/%s.yml", configDir, env),
		fmt.Sprintf("%s/%s.json", configDir, env),
		fmt.Sprintf("%s/config.yaml", configDir),
		fmt.Sprintf("%s/config.yml", configDir),
		fmt.Sprintf("%s/config.json", configDir),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// overrideWithEnvVars overrides config with specific environment variables
func overrideWithEnvVars(config *Config) {
	// Database connection info
	if val := os.Getenv("DATABASE_HOST"); val != "" {
		config.Database.Host = val
	}
	if val := os.Getenv("DATABASE_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Database.Port = port
		}
	}
	if val := os.Getenv("DATABASE_USERNAME"); val != "" {
		config.Database.Username = val
	}
	if val := os.Getenv("DATABASE_PASSWORD"); val != "" {
		config.Database.Password = val
	}
	if val := os.Getenv("DATABASE_NAME"); val != "" {
		config.Database.Database = val
	}

	// Cache connection info
	if val := os.Getenv("REDIS_HOST"); val != "" {
		config.Cache.Host = val
	}
	if val := os.Getenv("REDIS_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Cache.Port = port
		}
	}
	if val := os.Getenv("REDIS_PASSWORD"); val != "" {
		config.Cache.Password = val
	}

	// Kafka connection info
	if val := os.Getenv("KAFKA_BOOTSTRAP_SERVERS"); val != "" {
		config.EventBus.BootstrapServers = val
	}
	if val := os.Getenv("KAFKA_SASL_USERNAME"); val != "" {
		config.EventBus.SASLUsername = val
	}
	if val := os.Getenv("KAFKA_SASL_PASSWORD"); val != "" {
		config.EventBus.SASLPassword = val
	}

	// Server ports
	if val := os.Getenv("GRPC_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Server.GRPCPort = port
		}
	}
	if val := os.Getenv("HTTP_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Server.HTTPPort = port
		}
	}

	// JWT Secret
	if val := os.Getenv("JWT_SECRET"); val != "" {
		config.Security.JWTSecret = val
	}

	// Environment
	if val := os.Getenv("ENVIRONMENT"); val != "" {
		config.Environment = val
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate environment
	if config.Environment == "" {
		return fmt.Errorf("environment must be set")
	}

	// Validate server ports
	if config.Server.GRPCPort <= 0 {
		return fmt.Errorf("grpc port must be positive")
	}
	if config.Server.HTTPPort <= 0 {
		return fmt.Errorf("http port must be positive")
	}

	// Validate database config
	if config.Database.Host == "" {
		return fmt.Errorf("database host must be set")
	}
	if config.Database.Port <= 0 {
		return fmt.Errorf("database port must be positive")
	}
	if config.Database.Username == "" {
		return fmt.Errorf("database username must be set")
	}
	if config.Database.Database == "" {
		return fmt.Errorf("database name must be set")
	}

	// Validate Redis config
	if config.Cache.Host == "" {
		return fmt.Errorf("cache host must be set")
	}
	if config.Cache.Port <= 0 {
		return fmt.Errorf("cache port must be positive")
	}

	// Validate Kafka config
	if config.EventBus.BootstrapServers == "" {
		return fmt.Errorf("kafka bootstrap servers must be set")
	}

	// Validate security if TLS is enabled
	if config.Security.EnableTLS {
		if config.Security.TLSCertFile == "" {
			return fmt.Errorf("TLS cert file must be set when TLS is enabled")
		}
		if config.Security.TLSKeyFile == "" {
			return fmt.Errorf("TLS key file must be set when TLS is enabled")
		}
	}

	return nil
}
