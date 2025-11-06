package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save current env and restore after test
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			// Skip parsing errors
			if len(env) > 0 {
				for i := 0; i < len(env); i++ {
					if env[i] == '=' {
						os.Setenv(env[:i], env[i+1:])
						break
					}
				}
			}
		}
	}()

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.0.0.0", cfg.Server.Host)
				assert.Equal(t, 8080, cfg.Server.Port)
				assert.Equal(t, "localhost", cfg.Database.Host)
				assert.Equal(t, 5432, cfg.Database.Port)
				assert.Equal(t, "postgres", cfg.Database.User)
				assert.Equal(t, "workflows", cfg.Database.Database)
			},
		},
		{
			name: "custom values",
			env: map[string]string{
				"SERVER_PORT": "9000",
				"DB_HOST":     "db.example.com",
				"DB_PORT":     "5433",
				"DB_NAME":     "custom_db",
				"REDIS_HOST":  "redis.example.com",
				"REDIS_PORT":  "6380",
				"LOG_LEVEL":   "debug",
				"APP_ENV":     "production",
			},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 9000, cfg.Server.Port)
				assert.Equal(t, "db.example.com", cfg.Database.Host)
				assert.Equal(t, 5433, cfg.Database.Port)
				assert.Equal(t, "custom_db", cfg.Database.Database)
				assert.Equal(t, "redis.example.com", cfg.Redis.Host)
				assert.Equal(t, 6380, cfg.Redis.Port)
				assert.Equal(t, "debug", cfg.Logger.Level)
				assert.Equal(t, "production", cfg.App.Environment)
			},
		},
		{
			name: "invalid port",
			env: map[string]string{
				"SERVER_PORT": "99999",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set env vars
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg, err := Load()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					Database: "workflows",
				},
				Redis: RedisConfig{Host: "localhost"},
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Server: ServerConfig{Port: 0},
				Database: DatabaseConfig{
					Host:     "localhost",
					Database: "workflows",
				},
				Redis: RedisConfig{Host: "localhost"},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Server: ServerConfig{Port: 70000},
				Database: DatabaseConfig{
					Host:     "localhost",
					Database: "workflows",
				},
				Redis: RedisConfig{Host: "localhost"},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "missing database host",
			config: &Config{
				Server:   ServerConfig{Port: 8080},
				Database: DatabaseConfig{Database: "workflows"},
				Redis:    RedisConfig{Host: "localhost"},
			},
			wantErr: true,
			errMsg:  "database host is required",
		},
		{
			name: "missing database name",
			config: &Config{
				Server:   ServerConfig{Port: 8080},
				Database: DatabaseConfig{Host: "localhost"},
				Redis:    RedisConfig{Host: "localhost"},
			},
			wantErr: true,
			errMsg:  "database name is required",
		},
		{
			name: "missing redis host",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Database: DatabaseConfig{
					Host:     "localhost",
					Database: "workflows",
				},
			},
			wantErr: true,
			errMsg:  "redis host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_DatabaseDSN(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "db.example.com",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			Database: "testdb",
			SSLMode:  "require",
		},
	}

	dsn := cfg.DatabaseDSN()

	assert.Contains(t, dsn, "host=db.example.com")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=testuser")
	assert.Contains(t, dsn, "password=testpass")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=require")
}

func TestConfig_RedisAddr(t *testing.T) {
	cfg := &Config{
		Redis: RedisConfig{
			Host: "redis.example.com",
			Port: 6379,
		},
	}

	addr := cfg.RedisAddr()
	assert.Equal(t, "redis.example.com:6379", addr)
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "invalid integer",
			key:          "TEST_INT",
			envValue:     "not_a_number",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "empty value",
			key:          "TEST_INT",
			envValue:     "",
			defaultValue: 10,
			expected:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvAsInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "valid duration",
			key:          "TEST_DURATION",
			envValue:     "30s",
			defaultValue: 10 * time.Second,
			expected:     30 * time.Second,
		},
		{
			name:         "invalid duration",
			key:          "TEST_DURATION",
			envValue:     "not_a_duration",
			defaultValue: 10 * time.Second,
			expected:     10 * time.Second,
		},
		{
			name:         "empty value",
			key:          "TEST_DURATION",
			envValue:     "",
			defaultValue: 10 * time.Second,
			expected:     10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnvAsDuration(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
