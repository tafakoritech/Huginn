package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration, sourced from environment variables.
type Config struct {
	AppName    string
	Env        string
	APIPort    string
	WorkerName string

	Postgres PostgresConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	MinIO    MinIOConfig
	Tika     TikaConfig

	Auth AuthConfig

	LLM LLMConfig

	Storage StorageConfig

	Qualification QualificationConfig

	Messaging MessagingConfig

	// CallScenarioOperationTimeout bounds the ENTIRE call-scenario generation
	// operation (LLM attempt + retries + repair attempts + parsing) so it always
	// completes inside the HTTP server WriteTimeout. The hierarchy is deliberate:
	// LLM per-attempt (LLM_TIMEOUT_SECONDS, 180s) < operation (185s) < Go
	// WriteTimeout (240s) < client call timeout (250s).
	CallScenarioOperationTimeout time.Duration

	Muninn MuninnConfig

	Security      SecurityConfig
	Observability ObservabilityConfig

	Seed     SeedConfig
	Timezone string
}

type PostgresConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d pool_max_conn_lifetime=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode, p.MaxConns, p.MinConns, p.MaxConnLifetime,
	)
}

// PlainDSN is a simple DSN without pgxpool tuning parameters, valid for pgx.Connect.
func (p PostgresConfig) PlainDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type RabbitMQConfig struct {
	URL           string
	JobRetryLimit int
	JobRetryDelay time.Duration
}

type MinIOConfig struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	Bucket          string
	PresignTTLHours int
}

type TikaConfig struct {
	BaseURL string
}

type AuthConfig struct {
	AccessSecret      string
	RefreshSecret     string
	AccessTTL         time.Duration
	RefreshTTL        time.Duration
	Issuer            string
	Audience          string
	RefreshCookieName string
}

type LLMConfig struct {
	BaseURL        string
	APIKey         string
	Model          string
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
	PersistPrompts bool
	// StructuredMode controls which response_format the provider requests for
	// structured outputs: "auto" (probe json_schema, fall back to json_object),
	// "json_object" (always json_object — required for providers that reject
	// json_schema, e.g. DeepSeek's deepseek-v4-flash returns HTTP 400
	// "This response_format type is unavailable now"), or "json_schema".
	StructuredMode string
	// SOCKS5Proxy is an optional "host:port" SOCKS5 proxy for OUTBOUND LLM
	// traffic only. Empty means direct connection. This is never routed through
	// the generic HTTP_PROXY/HTTPS_PROXY/ALL_PROXY machinery; it is applied
	// explicitly to the LLM HTTP transport.
	SOCKS5Proxy string
}

type StorageConfig struct {
	MaxUploadBytes int64
}

type QualificationConfig struct {
	ScoreThreshold int
}

type MuninnConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// MessagingConfig describes OUR company and the sender profile. This identity
// belongs to the salesperson sending outreach, NEVER to the lead/recipient.
type MessagingConfig struct {
	CompanyName        string
	CompanyDescription string
	SenderName         string
	SenderTitle        string
}

type SecurityConfig struct {
	AllowedOrigins  []string
	RateLimitPerMin int
}

type ObservabilityConfig struct {
	MetricsEnabled bool
}

type SeedConfig struct {
	AdminPassword   string
	ManagerPassword string
	AgentPassword   string
}

func Load() (*Config, error) {
	c := &Config{}

	c.AppName = getEnv("APP_NAME", "huginn")
	c.Env = getEnv("APP_ENV", "development")
	c.APIPort = getEnv("HTTP_PORT", "8080")
	c.WorkerName = getEnv("WORKER_NAME", "worker-1")
	c.Timezone = getEnv("APP_TIMEZONE", "Asia/Tehran")

	c.Postgres = PostgresConfig{
		Host:            getEnv("POSTGRES_HOST", "localhost"),
		Port:            getEnv("POSTGRES_PORT", "5432"),
		User:            getEnv("POSTGRES_USER", "huginn"),
		Password:        getEnv("POSTGRES_PASSWORD", "huginn"),
		DBName:          getEnv("POSTGRES_DB", "huginn"),
		SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
		MaxConns:        getEnvInt("POSTGRES_MAX_CONNS", 20),
		MinConns:        getEnvInt("POSTGRES_MIN_CONNS", 2),
		MaxConnLifetime: time.Duration(getEnvInt("POSTGRES_MAX_CONN_LIFETIME", 30)) * time.Minute,
	}

	c.Redis = RedisConfig{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
	}

	c.RabbitMQ = RabbitMQConfig{
		URL:           getEnv("RABBITMQ_URL", "amqp://huginn:huginn@localhost:5672/"),
		JobRetryLimit: getEnvInt("RABBITMQ_JOB_RETRY_LIMIT", 5),
		JobRetryDelay: time.Duration(getEnvInt("RABBITMQ_JOB_RETRY_DELAY_MS", 5000)) * time.Millisecond,
	}

	c.MinIO = MinIOConfig{
		Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKey:       getEnv("MINIO_ACCESS_KEY", "huginn"),
		SecretKey:       getEnv("MINIO_SECRET_KEY", "huginnsecret"),
		UseSSL:          getEnvBool("MINIO_USE_SSL", false),
		Bucket:          getEnv("MINIO_BUCKET", "huginn-source-documents"),
		PresignTTLHours: getEnvInt("MINIO_PRESIGN_TTL_HOURS", 1),
	}

	c.Tika = TikaConfig{
		BaseURL: getEnv("TIKA_BASE_URL", "http://localhost:9998"),
	}

	c.Auth = AuthConfig{
		AccessSecret:      getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		RefreshSecret:     getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
		AccessTTL:         time.Duration(getEnvInt("JWT_ACCESS_TTL_MINUTES", 15)) * time.Minute,
		RefreshTTL:        time.Duration(getEnvInt("JWT_REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,
		Issuer:            getEnv("JWT_ISSUER", "huginn"),
		Audience:          getEnv("JWT_AUDIENCE", "huginn-clients"),
		RefreshCookieName: getEnv("REFRESH_COOKIE_NAME", "huginn_refresh"),
	}

	c.LLM = LLMConfig{
		BaseURL:        getEnv("LLM_BASE_URL", ""),
		APIKey:         getEnv("LLM_API_KEY", ""),
		Model:          getEnv("LLM_MODEL", "deepseek-v4-flash"),
		Timeout:        time.Duration(getEnvInt("LLM_TIMEOUT_SECONDS", 60)) * time.Second,
		MaxRetries:     getEnvInt("LLM_MAX_RETRIES", 3),
		RetryBaseDelay: time.Duration(getEnvInt("LLM_RETRY_BASE_DELAY_MS", 500)) * time.Millisecond,
		PersistPrompts: getEnvBool("LLM_PERSIST_PROMPTS", false),
		// DeepSeek's deepseek-v4-flash rejects response_format json_schema with
		// HTTP 400 ("This response_format type is unavailable now"), so the
		// deployment pins json_object mode. Set to "auto" to probe json_schema
		// first (safe only for providers that actually support it).
		StructuredMode: getEnv("LLM_STRUCTURED_OUTPUT_MODE", "auto"),
		// Dedicated SOCKS5 proxy for LLM outbound traffic only. Empty = direct.
		// See docker-compose.yml: LLM_SOCKS5_PROXY overrides this for containers.
		SOCKS5Proxy: getEnv("LLM_SOCKS5_PROXY", ""),
	}

	c.Storage = StorageConfig{
		MaxUploadBytes: int64(getEnvInt("MAX_UPLOAD_BYTES", 25*1024*1024)),
	}

	c.Qualification = QualificationConfig{
		ScoreThreshold: getEnvInt("QUALIFICATION_SCORE_THRESHOLD", 70),
	}

	c.Messaging = MessagingConfig{
		CompanyName:        getEnv("MESSAGING_COMPANY_NAME", ""),
		CompanyDescription: getEnv("MESSAGING_COMPANY_DESCRIPTION", ""),
		SenderName:         getEnv("MESSAGING_SENDER_NAME", ""),
		SenderTitle:        getEnv("MESSAGING_SENDER_TITLE", ""),
	}

	// The whole call-scenario operation is bounded so the handler always writes
	// its response before the HTTP server WriteTimeout fires. This must be a few
	// seconds LONGER than LLM_TIMEOUT_SECONDS (one full attempt) but comfortably
	// SHORTER than the server WriteTimeout.
	c.CallScenarioOperationTimeout = time.Duration(getEnvInt("CALL_SCENARIO_OP_TIMEOUT_SECONDS", 185)) * time.Second

	c.Muninn = MuninnConfig{
		Enabled: getEnvBool("MUNINN_ENABLED", false),
		BaseURL: getEnv("MUNINN_BASE_URL", ""),
		APIKey:  getEnv("MUNINN_API_KEY", ""),
		Timeout: time.Duration(getEnvInt("MUNINN_TIMEOUT_SECONDS", 30)) * time.Second,
	}

	c.Security = SecurityConfig{
		AllowedOrigins:  strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
		RateLimitPerMin: getEnvInt("RATE_LIMIT_PER_MIN", 300),
	}

	c.Observability = ObservabilityConfig{
		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
	}

	c.Seed = SeedConfig{
		AdminPassword:   getEnv("SEED_ADMIN_PASSWORD", "Admin@12345"),
		ManagerPassword: getEnv("SEED_MANAGER_PASSWORD", "Manager@12345"),
		AgentPassword:   getEnv("SEED_AGENT_PASSWORD", "Agent@12345"),
	}

	return c, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
