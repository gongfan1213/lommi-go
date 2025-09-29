package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	App                     AppConfig                     `json:"app" yaml:"app"`
	API                     APIConfig                     `json:"api" yaml:"api"`
	LLM                     LLMConfig                     `json:"llm" yaml:"llm"`
	Security                SecurityConfig                `json:"security" yaml:"security"`
	Memory                  MemoryConfig                  `json:"memory" yaml:"memory"`
	Database                DatabaseConfig                `json:"database" yaml:"database"`
	Performance             PerformanceConfig             `json:"performance" yaml:"performance"`
	Monitoring              MonitoringConfig              `json:"monitoring" yaml:"monitoring"`
	LangSmith               LangSmithConfig               `json:"langsmith" yaml:"langsmith"`
	Nova3                   Nova3Config                   `json:"nova3" yaml:"nova3"`
	Dashboard               DashboardConfig               `json:"dashboard" yaml:"dashboard"`
	Gemini                  GeminiConfig                  `json:"gemini" yaml:"gemini"`
	LoomiRevision           LoomiRevisionConfig           `json:"loomi_revision" yaml:"loomi_revision"`
	PerformanceOptimization PerformanceOptimizationConfig `json:"performance_optimization" yaml:"performance_optimization"`
}

// AppConfig represents application configuration
type AppConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Debug       bool   `json:"debug"`
	LogLevel    string `json:"log_level"`
	Environment string `json:"environment"`
}

// APIConfig represents API configuration
type APIConfig struct {
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	CORSOrigins    []string `json:"cors_origins"`
	MaxRequestSize int64    `json:"max_request_size"`
	Timeout        int      `json:"timeout"`
}

// LLMConfig represents LLM configuration
type LLMConfig struct {
	DefaultProvider string                       `json:"default_provider"`
	Providers       map[string]LLMProviderConfig `json:"providers"`
}

// LLMProviderConfig represents LLM provider configuration
type LLMProviderConfig struct {
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"base_url"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	JWTSecretKey             string   `json:"jwt_secret_key"`
	AccessTokenExpireMinutes int      `json:"access_token_expire_minutes"`
	RateLimitPerMinute       int      `json:"rate_limit_per_minute"`
	EnableAuth               bool     `json:"enable_auth"`
	Whitelist                []string `json:"whitelist"`
}

// MemoryConfig represents memory storage configuration
type MemoryConfig struct {
	StoreType     string `json:"store_type"`
	RedisHost     string `json:"redis_host"`
	RedisPort     int    `json:"redis_port"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	SupabaseURL    string `json:"supabase_url" yaml:"supabase_url"`
	SupabaseKey    string `json:"supabase_key" yaml:"supabase_key"`
	SupabaseSecret string `json:"supabase_secret" yaml:"supabase_secret"`
}

// PerformanceConfig represents performance configuration
type PerformanceConfig struct {
	EnableMetrics     bool `json:"enable_metrics" yaml:"enable_metrics"`
	EnableRateLimit   bool `json:"enable_rate_limit" yaml:"enable_rate_limit"`
	MaxConcurrency    int  `json:"max_concurrency" yaml:"max_concurrency"`
	RequestTimeout    int  `json:"request_timeout" yaml:"request_timeout"`
	ConnectionTimeout int  `json:"connection_timeout" yaml:"connection_timeout"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	EnableMetrics        bool   `json:"enable_metrics" yaml:"enable_metrics"`
	MetricsPort          int    `json:"metrics_port" yaml:"metrics_port"`
	HealthCheckInterval  int    `json:"health_check_interval" yaml:"health_check_interval"`
	AlertWebhookURL      string `json:"alert_webhook_url" yaml:"alert_webhook_url"`
	EnablePortMonitoring bool   `json:"enable_port_monitoring" yaml:"enable_port_monitoring"`
	MonitorHost          string `json:"monitor_host" yaml:"monitor_host"`
}

// LangSmithConfig represents LangSmith configuration
type LangSmithConfig struct {
	APIKey        string `json:"api_key" yaml:"api_key"`
	Project       string `json:"project" yaml:"project"`
	Endpoint      string `json:"endpoint" yaml:"endpoint"`
	EnableTracing bool   `json:"enable_tracing" yaml:"enable_tracing"`
}

// Nova3Config represents Nova3 configuration
type Nova3Config struct {
	QueueManagerConfig QueueManagerConfig `json:"queue_manager" yaml:"queue_manager"`
	AgentConfig        AgentConfig        `json:"agent" yaml:"agent"`
}

// QueueManagerConfig represents queue manager configuration
type QueueManagerConfig struct {
	MaxQueueSize      int `json:"max_queue_size" yaml:"max_queue_size"`
	ProcessingTimeout int `json:"processing_timeout" yaml:"processing_timeout"`
	RetryAttempts     int `json:"retry_attempts" yaml:"retry_attempts"`
	CleanupInterval   int `json:"cleanup_interval" yaml:"cleanup_interval"`
}

// AgentConfig represents agent configuration
type AgentConfig struct {
	MaxConcurrentAgents int  `json:"max_concurrent_agents" yaml:"max_concurrent_agents"`
	AgentTimeout        int  `json:"agent_timeout" yaml:"agent_timeout"`
	EnableParallelMode  bool `json:"enable_parallel_mode" yaml:"enable_parallel_mode"`
}

// DashboardConfig represents dashboard configuration
type DashboardConfig struct {
	EnableDashboard bool   `json:"enable_dashboard" yaml:"enable_dashboard"`
	DashboardPort   int    `json:"dashboard_port" yaml:"dashboard_port"`
	RefreshInterval int    `json:"refresh_interval" yaml:"refresh_interval"`
	Theme           string `json:"theme" yaml:"theme"`
}

// GeminiConfig represents Gemini configuration
type GeminiConfig struct {
	APIKey            string  `json:"api_key" yaml:"api_key"`
	Model             string  `json:"model" yaml:"model"`
	Temperature       float64 `json:"temperature" yaml:"temperature"`
	MaxTokens         int     `json:"max_tokens" yaml:"max_tokens"`
	EnableMultimodal  bool    `json:"enable_multimodal" yaml:"enable_multimodal"`
	MultimodalTimeout int     `json:"multimodal_timeout" yaml:"multimodal_timeout"`
}

// LoomiRevisionConfig represents Loomi revision configuration
type LoomiRevisionConfig struct {
	EnableRevision     bool `json:"enable_revision" yaml:"enable_revision"`
	MaxRevisionHistory int  `json:"max_revision_history" yaml:"max_revision_history"`
	AutoSave           bool `json:"auto_save" yaml:"auto_save"`
	SaveInterval       int  `json:"save_interval" yaml:"save_interval"`
}

// PerformanceOptimizationConfig represents performance optimization configuration
type PerformanceOptimizationConfig struct {
	EnableConnectionPooling bool `json:"enable_connection_pooling" yaml:"enable_connection_pooling"`
	PoolSize                int  `json:"pool_size" yaml:"pool_size"`
	MaxIdleConnections      int  `json:"max_idle_connections" yaml:"max_idle_connections"`
	ConnectionTimeout       int  `json:"connection_timeout" yaml:"connection_timeout"`
	EnableCaching           bool `json:"enable_caching" yaml:"enable_caching"`
	CacheTTL                int  `json:"cache_ttl" yaml:"cache_ttl"`
}

// Load loads configuration from YAML files and environment variables
func Load() *Config {
	config := &Config{}

	// Load YAML configuration first
	configDir := getEnv("CONFIG_DIR", "config")
	yamlConfig := loadYAMLConfig(configDir)

	// Load app configuration
	config.App = AppConfig{
		Name:        getEnvWithYAML("APP_NAME", yamlConfig, "app.name", "BluePlan Research"),
		Version:     getEnvWithYAML("APP_VERSION", yamlConfig, "app.version", "1.0.0"),
		Debug:       getEnvBoolWithYAML("DEBUG", yamlConfig, "app.debug", true),
		LogLevel:    getEnvWithYAML("LOG_LEVEL", yamlConfig, "app.log_level", "INFO"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	// Load API configuration
	config.API = APIConfig{
		Host:           getEnvWithYAML("API_HOST", yamlConfig, "api.host", "0.0.0.0"),
		Port:           getEnvIntWithYAML("API_PORT", yamlConfig, "api.port", 8000),
		CORSOrigins:    getEnvSliceWithYAML("API_CORS_ORIGINS", yamlConfig, "api.cors_origins", []string{"*"}),
		MaxRequestSize: getEnvInt64WithYAML("MAX_REQUEST_SIZE", yamlConfig, "api.max_request_size", 10485760),
		Timeout:        getEnvIntWithYAML("API_TIMEOUT", yamlConfig, "api.timeout", 300),
	}

	// Load LLM configuration
	config.LLM = LLMConfig{
		DefaultProvider: getEnvWithYAML("LLM_DEFAULT_PROVIDER", yamlConfig, "llm.default_provider", "openai"),
		Providers:       loadLLMProviders(),
	}

	// Load Security configuration
	config.Security = SecurityConfig{
		JWTSecretKey:             getEnvWithYAML("JWT_SECRET_KEY", yamlConfig, "security.jwt_secret_key", "default_secret_key"),
		AccessTokenExpireMinutes: getEnvIntWithYAML("ACCESS_TOKEN_EXPIRE_MINUTES", yamlConfig, "security.access_token_expire_minutes", 30),
		RateLimitPerMinute:       getEnvIntWithYAML("RATE_LIMIT_PER_MINUTE", yamlConfig, "security.rate_limit_per_minute", 60),
		EnableAuth:               getEnvBoolWithYAML("ENABLE_AUTH", yamlConfig, "security.enable_auth", false),
		Whitelist:                getEnvSliceWithYAML("API_WHITELIST", yamlConfig, "security.whitelist", []string{"/health"}),
	}

	// Load Memory configuration
	config.Memory = MemoryConfig{
		StoreType:     getEnvWithYAML("MEMORY_STORE_TYPE", yamlConfig, "memory.store_type", "memory"),
		RedisHost:     getEnvWithYAML("REDIS_HOST", yamlConfig, "memory.redis_host", "localhost"),
		RedisPort:     getEnvIntWithYAML("REDIS_PORT", yamlConfig, "memory.redis_port", 6379),
		RedisPassword: getEnvWithYAML("REDIS_PASSWORD", yamlConfig, "memory.redis_password", ""),
		RedisDB:       getEnvIntWithYAML("REDIS_DB", yamlConfig, "memory.redis_db", 0),
	}

	// Load Database configuration
	config.Database = DatabaseConfig{
		SupabaseURL:    getEnvWithYAML("SUPABASE_URL", yamlConfig, "database.supabase_url", ""),
		SupabaseKey:    getEnvWithYAML("SUPABASE_KEY", yamlConfig, "database.supabase_key", ""),
		SupabaseSecret: getEnvWithYAML("SUPABASE_SECRET", yamlConfig, "database.supabase_secret", ""),
	}

	// Load Performance configuration
	config.Performance = PerformanceConfig{
		EnableMetrics:     getEnvBoolWithYAML("ENABLE_METRICS", yamlConfig, "performance.enable_metrics", true),
		EnableRateLimit:   getEnvBoolWithYAML("ENABLE_RATE_LIMIT", yamlConfig, "performance.enable_rate_limit", false),
		MaxConcurrency:    getEnvIntWithYAML("MAX_CONCURRENCY", yamlConfig, "performance.max_concurrency", 100),
		RequestTimeout:    getEnvIntWithYAML("REQUEST_TIMEOUT", yamlConfig, "performance.request_timeout", 300),
		ConnectionTimeout: getEnvIntWithYAML("CONNECTION_TIMEOUT", yamlConfig, "performance.connection_timeout", 30),
	}

	// Load Monitoring configuration
	config.Monitoring = MonitoringConfig{
		EnableMetrics:        getEnvBoolWithYAML("ENABLE_METRICS", yamlConfig, "monitoring.enable_metrics", false),
		MetricsPort:          getEnvIntWithYAML("METRICS_PORT", yamlConfig, "monitoring.metrics_port", 9090),
		HealthCheckInterval:  getEnvIntWithYAML("HEALTH_CHECK_INTERVAL", yamlConfig, "monitoring.health_check_interval", 60),
		AlertWebhookURL:      getEnvWithYAML("ALERT_WEBHOOK_URL", yamlConfig, "monitoring.alert_webhook_url", ""),
		EnablePortMonitoring: getEnvBoolWithYAML("ENABLE_PORT_MONITORING", yamlConfig, "monitoring.enable_port_monitoring", false),
		MonitorHost:          getEnvWithYAML("MONITOR_HOST", yamlConfig, "monitoring.monitor_host", "localhost"),
	}

	// Load LangSmith configuration
	config.LangSmith = LangSmithConfig{
		APIKey:        getEnvWithYAML("LANGSMITH_API_KEY", yamlConfig, "langsmith.api_key", ""),
		Project:       getEnvWithYAML("LANGSMITH_PROJECT", yamlConfig, "langsmith.project", ""),
		Endpoint:      getEnvWithYAML("LANGSMITH_ENDPOINT", yamlConfig, "langsmith.endpoint", ""),
		EnableTracing: getEnvBoolWithYAML("LANGSMITH_ENABLE_TRACING", yamlConfig, "langsmith.enable_tracing", false),
	}

	// Load Nova3 configuration
	config.Nova3 = Nova3Config{
		QueueManagerConfig: QueueManagerConfig{
			MaxQueueSize:      getEnvIntWithYAML("NOVA3_MAX_QUEUE_SIZE", yamlConfig, "nova3.queue_manager.max_queue_size", 1000),
			ProcessingTimeout: getEnvIntWithYAML("NOVA3_PROCESSING_TIMEOUT", yamlConfig, "nova3.queue_manager.processing_timeout", 300),
			RetryAttempts:     getEnvIntWithYAML("NOVA3_RETRY_ATTEMPTS", yamlConfig, "nova3.queue_manager.retry_attempts", 3),
			CleanupInterval:   getEnvIntWithYAML("NOVA3_CLEANUP_INTERVAL", yamlConfig, "nova3.queue_manager.cleanup_interval", 3600),
		},
		AgentConfig: AgentConfig{
			MaxConcurrentAgents: getEnvIntWithYAML("NOVA3_MAX_CONCURRENT_AGENTS", yamlConfig, "nova3.agent.max_concurrent_agents", 10),
			AgentTimeout:        getEnvIntWithYAML("NOVA3_AGENT_TIMEOUT", yamlConfig, "nova3.agent.agent_timeout", 60),
			EnableParallelMode:  getEnvBoolWithYAML("NOVA3_ENABLE_PARALLEL_MODE", yamlConfig, "nova3.agent.enable_parallel_mode", true),
		},
	}

	// Load Dashboard configuration
	config.Dashboard = DashboardConfig{
		EnableDashboard: getEnvBoolWithYAML("ENABLE_DASHBOARD", yamlConfig, "dashboard.enable_dashboard", true),
		DashboardPort:   getEnvIntWithYAML("DASHBOARD_PORT", yamlConfig, "dashboard.dashboard_port", 8080),
		RefreshInterval: getEnvIntWithYAML("DASHBOARD_REFRESH_INTERVAL", yamlConfig, "dashboard.refresh_interval", 30),
		Theme:           getEnvWithYAML("DASHBOARD_THEME", yamlConfig, "dashboard.theme", "light"),
	}

	// Load Gemini configuration
	config.Gemini = GeminiConfig{
		APIKey:            getEnvWithYAML("GEMINI_API_KEY", yamlConfig, "gemini.api_key", ""),
		Model:             getEnvWithYAML("GEMINI_MODEL", yamlConfig, "gemini.model", "gemini-1.5-flash"),
		Temperature:       getEnvFloat64WithYAML("GEMINI_TEMPERATURE", yamlConfig, "gemini.temperature", 0.7),
		MaxTokens:         getEnvIntWithYAML("GEMINI_MAX_TOKENS", yamlConfig, "gemini.max_tokens", 4096),
		EnableMultimodal:  getEnvBoolWithYAML("GEMINI_ENABLE_MULTIMODAL", yamlConfig, "gemini.enable_multimodal", true),
		MultimodalTimeout: getEnvIntWithYAML("GEMINI_MULTIMODAL_TIMEOUT", yamlConfig, "gemini.multimodal_timeout", 60),
	}

	// Load LoomiRevision configuration
	config.LoomiRevision = LoomiRevisionConfig{
		EnableRevision:     getEnvBoolWithYAML("LOOMI_ENABLE_REVISION", yamlConfig, "loomi_revision.enable_revision", true),
		MaxRevisionHistory: getEnvIntWithYAML("LOOMI_MAX_REVISION_HISTORY", yamlConfig, "loomi_revision.max_revision_history", 10),
		AutoSave:           getEnvBoolWithYAML("LOOMI_AUTO_SAVE", yamlConfig, "loomi_revision.auto_save", true),
		SaveInterval:       getEnvIntWithYAML("LOOMI_SAVE_INTERVAL", yamlConfig, "loomi_revision.save_interval", 30),
	}

	// Load PerformanceOptimization configuration
	config.PerformanceOptimization = PerformanceOptimizationConfig{
		EnableConnectionPooling: getEnvBoolWithYAML("ENABLE_CONNECTION_POOLING", yamlConfig, "performance_optimization.enable_connection_pooling", true),
		PoolSize:                getEnvIntWithYAML("POOL_SIZE", yamlConfig, "performance_optimization.pool_size", 100),
		MaxIdleConnections:      getEnvIntWithYAML("MAX_IDLE_CONNECTIONS", yamlConfig, "performance_optimization.max_idle_connections", 50),
		ConnectionTimeout:       getEnvIntWithYAML("CONNECTION_TIMEOUT", yamlConfig, "performance_optimization.connection_timeout", 30),
		EnableCaching:           getEnvBoolWithYAML("ENABLE_CACHING", yamlConfig, "performance_optimization.enable_caching", true),
		CacheTTL:                getEnvIntWithYAML("CACHE_TTL", yamlConfig, "performance_optimization.cache_ttl", 3600),
	}

	return config
}

// loadLLMProviders loads LLM provider configurations
func loadLLMProviders() map[string]LLMProviderConfig {
	providers := make(map[string]LLMProviderConfig)

	// OpenAI provider
	if apiKey := getEnv("OPENAI_API_KEY", ""); apiKey != "" {
		providers["openai"] = LLMProviderConfig{
			APIKey:      apiKey,
			Model:       getEnv("OPENAI_MODEL", "gpt-4"),
			BaseURL:     getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			Temperature: getEnvFloat64("OPENAI_TEMPERATURE", 0.7),
			MaxTokens:   getEnvInt("OPENAI_MAX_TOKENS", 4096),
		}
	}

	// Claude provider
	if apiKey := getEnv("CLAUDE_API_KEY", ""); apiKey != "" {
		providers["claude"] = LLMProviderConfig{
			APIKey:      apiKey,
			Model:       getEnv("CLAUDE_MODEL", "claude-3-sonnet-20240229"),
			BaseURL:     getEnv("CLAUDE_BASE_URL", "https://api.anthropic.com"),
			Temperature: getEnvFloat64("CLAUDE_TEMPERATURE", 0.7),
			MaxTokens:   getEnvInt("CLAUDE_MAX_TOKENS", 4096),
		}
	}

	// Gemini provider
	if apiKey := getEnv("GEMINI_API_KEY", ""); apiKey != "" {
		providers["gemini"] = LLMProviderConfig{
			APIKey:      apiKey,
			Model:       getEnv("GEMINI_MODEL", "gemini-1.5-flash"),
			BaseURL:     getEnv("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta"),
			Temperature: getEnvFloat64("GEMINI_TEMPERATURE", 0.7),
			MaxTokens:   getEnvInt("GEMINI_MAX_TOKENS", 4096),
		}
	}

	return providers
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// loadYAMLConfig loads configuration from YAML files
func loadYAMLConfig(configDir string) map[string]interface{} {
	yamlConfig := make(map[string]interface{})

	// Try to load app_config.yaml
	appConfigPath := filepath.Join(configDir, "app_config.yaml")
	if data, err := os.ReadFile(appConfigPath); err == nil {
		var config map[string]interface{}
		if err := yaml.Unmarshal(data, &config); err == nil {
			yamlConfig = config
		}
	}

	// Try to load llm_config.yaml
	llmConfigPath := filepath.Join(configDir, "llm_config.yaml")
	if data, err := os.ReadFile(llmConfigPath); err == nil {
		var llmConfig map[string]interface{}
		if err := yaml.Unmarshal(data, &llmConfig); err == nil {
			yamlConfig["llm"] = llmConfig
		}
	}

	return yamlConfig
}

// getEnvWithYAML gets environment variable with YAML fallback
func getEnvWithYAML(envKey string, yamlConfig map[string]interface{}, yamlPath, defaultValue string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}

	if yamlValue := getYAMLValue(yamlConfig, yamlPath); yamlValue != "" {
		return yamlValue
	}

	return defaultValue
}

// getEnvIntWithYAML gets integer environment variable with YAML fallback
func getEnvIntWithYAML(envKey string, yamlConfig map[string]interface{}, yamlPath string, defaultValue int) int {
	if value := os.Getenv(envKey); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	if yamlValue := getYAMLValue(yamlConfig, yamlPath); yamlValue != "" {
		if intValue, err := strconv.Atoi(yamlValue); err == nil {
			return intValue
		}
	}

	return defaultValue
}

// getEnvInt64WithYAML gets int64 environment variable with YAML fallback
func getEnvInt64WithYAML(envKey string, yamlConfig map[string]interface{}, yamlPath string, defaultValue int64) int64 {
	if value := os.Getenv(envKey); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}

	if yamlValue := getYAMLValue(yamlConfig, yamlPath); yamlValue != "" {
		if intValue, err := strconv.ParseInt(yamlValue, 10, 64); err == nil {
			return intValue
		}
	}

	return defaultValue
}

// getEnvFloat64WithYAML gets float64 environment variable with YAML fallback
func getEnvFloat64WithYAML(envKey string, yamlConfig map[string]interface{}, yamlPath string, defaultValue float64) float64 {
	if value := os.Getenv(envKey); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}

	if yamlValue := getYAMLValue(yamlConfig, yamlPath); yamlValue != "" {
		if floatValue, err := strconv.ParseFloat(yamlValue, 64); err == nil {
			return floatValue
		}
	}

	return defaultValue
}

// getEnvBoolWithYAML gets boolean environment variable with YAML fallback
func getEnvBoolWithYAML(envKey string, yamlConfig map[string]interface{}, yamlPath string, defaultValue bool) bool {
	if value := os.Getenv(envKey); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}

	if yamlValue := getYAMLValue(yamlConfig, yamlPath); yamlValue != "" {
		if boolValue, err := strconv.ParseBool(yamlValue); err == nil {
			return boolValue
		}
	}

	return defaultValue
}

// getEnvSliceWithYAML gets string slice environment variable with YAML fallback
func getEnvSliceWithYAML(envKey string, yamlConfig map[string]interface{}, yamlPath string, defaultValue []string) []string {
	if value := os.Getenv(envKey); value != "" {
		// Split by comma for environment variable
		parts := strings.Split(value, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result
	}

	if yamlValue := getYAMLSlice(yamlConfig, yamlPath); yamlValue != nil {
		return yamlValue
	}

	return defaultValue
}

// getYAMLValue gets value from YAML config using dot notation path
func getYAMLValue(config map[string]interface{}, path string) string {
	parts := strings.Split(path, ".")
	current := config

	for i, part := range parts {
		if i == len(parts)-1 {
			if value, ok := current[part]; ok {
				if str, ok := value.(string); ok {
					return str
				}
			}
			break
		}

		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			break
		}
	}

	return ""
}

// getYAMLSlice gets string slice from YAML config using dot notation path
func getYAMLSlice(config map[string]interface{}, path string) []string {
	parts := strings.Split(path, ".")
	current := config

	for i, part := range parts {
		if i == len(parts)-1 {
			if value, ok := current[part]; ok {
				if slice, ok := value.([]interface{}); ok {
					result := make([]string, len(slice))
					for i, item := range slice {
						if str, ok := item.(string); ok {
							result[i] = str
						}
					}
					return result
				}
			}
			break
		}

		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			break
		}
	}

	return nil
}
