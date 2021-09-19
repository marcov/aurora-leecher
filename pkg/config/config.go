package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"localhost/aurora/pkg/types"
)

type Config struct {
	Aurora types.AuroraConfig
	Email  types.EmailConfig
}

func (cfg *Config) FromFile(configFile string) error {
	rawConfig, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %q: %w", configFile, err)
	}

	if err := json.Unmarshal(rawConfig, cfg); err != nil {
		return fmt.Errorf("failed to json unmarshal: %w", err)
	}

	return nil
}

func (cfg *Config) FromEnvs() error {
	var (
		env   string
		value string
		set   bool
		err   error
	)

	env = "AURORA_USERNAME"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Aurora.Username = value

	env = "AURORA_PASSWORD"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Aurora.Password = value

	env = "AURORA_USERID"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Aurora.UserId, err = strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("env %q is not an int: %q", env, value)
	}

	env = "AURORA_ACTIVITYID"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Aurora.ActivityId, err = strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("env %q is not an int: %q", env, value)
	}

	env = "EMAIL_DOMAIN"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Email.Domain = value

	env = "EMAIL_APIKEY"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Email.ApiKey = value

	env = "EMAIL_FROM"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Email.From = value

	env = "EMAIL_TO"
	if value, set = os.LookupEnv(env); !set {
		return fmt.Errorf("env %q is not set", env)
	}
	cfg.Email.To = strings.Split(value, ",")

	return nil
}
