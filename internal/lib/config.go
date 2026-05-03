package lib

import (
	"errors"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
)

type JwtData struct {
	Id string `json:"id"`
}

type conf struct {
	DatabaseDSN            string // DSN link for connect with database
	JWTSecret              string
	TbankTerminalKey       string
	TbankTerminalPassword  string
	CloudPaymentsPublicID  string
	CloudPaymentsAPISecret string
	CloudPaymentsBaseURL   string
	TelegramToken          string // Optional: used to proxy Telegram avatars
	MyURL                  string
	FrontURL               string
	RedisHost              string
	RedisPort              string
	RedisPassword          string
	RedisDB                int
	// AI and Storage
	AIProvider       string // AI provider to use (perplexity, openai, auto)
	PerplexityAPIKey string
	OpenAIAPIKey     string
	S3Endpoint       string
	S3PublicBaseURL  string
	S3AccessKey      string
	S3SecretKey      string
	S3BucketName     string
	S3UseSSL         bool
	S3Region         string
	// Service-to-service auth
	ServiceToken string // Token for internal service communication (Telegram bot, etc.)
}

type contextKeys struct {
	UserKey     string
	UserIdKey   string
	TypeKey     string
	InitDataKey string
	TokenKey    string
	JwtDataKey  string
	FoundToken  string
}

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[cnfg]",
		Level:  log.DebugLevel,
	})

	Config      = &conf{}
	ContextKeys = &contextKeys{
		UserKey:     "user",
		UserIdKey:   "userId",
		TypeKey:     "type",
		InitDataKey: "initData",
		TokenKey:    "token",
		JwtDataKey:  "jwtData",
		FoundToken:  "foundToken",
	}
)

func ParseENV() error {
	if env, exist := os.LookupEnv("DATABASE_DSN"); exist {
		Config.DatabaseDSN = env
	} else {
		logger.Error("cannot find env DATABASE_DSN")
		return errors.New("cannot find env DATABASE_DSN")
	}

	if env, exist := os.LookupEnv("JWT_SECRET"); exist {
		Config.JWTSecret = env
	} else {
		logger.Error("cannot find env JWT_SECRET")
		return errors.New("cannot find env JWT_SECRET")
	}

	if env, exist := os.LookupEnv("TBANK_TERMINAL_KEY"); exist {
		Config.TbankTerminalKey = env
	} else {
		logger.Error("cannot find env TBANK_TERMINAL_KEY")
		return errors.New("cannot find env TBANK_TERMINAL_KEY")
	}

	if env, exist := os.LookupEnv("TBANK_TERMINAL_PASSWORD"); exist {
		Config.TbankTerminalPassword = env
	} else {
		logger.Error("cannot find env TBANK_TERMINAL_PASSWORD")
		return errors.New("cannot find env TBANK_TERMINAL_PASSWORD")
	}

	if env, exist := os.LookupEnv("CLOUDPAYMENTS_PUBLIC_ID"); exist {
		Config.CloudPaymentsPublicID = env
	} else {
		logger.Error("cannot find env CLOUDPAYMENTS_PUBLIC_ID")
		return errors.New("cannot find env CLOUDPAYMENTS_PUBLIC_ID")
	}

	if env, exist := os.LookupEnv("CLOUDPAYMENTS_API_SECRET"); exist {
		Config.CloudPaymentsAPISecret = env
	} else {
		logger.Error("cannot find env CLOUDPAYMENTS_API_SECRET")
		return errors.New("cannot find env CLOUDPAYMENTS_API_SECRET")
	}

	if env, exist := os.LookupEnv("CLOUDPAYMENTS_BASE_URL"); exist {
		Config.CloudPaymentsBaseURL = env
	} else {
		Config.CloudPaymentsBaseURL = "https://api.cloudpayments.ru"
	}

	if env, exist := os.LookupEnv("MY_URL"); exist {
		Config.MyURL = env
	} else {
		Config.MyURL = "https://api.nutri02.com"
	}

	if env, exist := os.LookupEnv("FRONT_URL"); exist {
		Config.FrontURL = env
	} else {
		Config.FrontURL = "https://nutri02.com"
	}

	if env, exist := os.LookupEnv("TELEGRAM_TOKEN"); exist {
		Config.TelegramToken = env
	} else {
		logger.Error("cannot find env TELEGRAM_TOKEN")
		return errors.New("cannot find env TELEGRAM_TOKEN")
	}

	if env, exist := os.LookupEnv("REDIS_HOST"); exist {
		Config.RedisHost = env
	} else {
		logger.Error("cannot find env REDIS_HOST")
		return errors.New("cannot find env REDIS_HOST")
	}

	if env, exist := os.LookupEnv("REDIS_PORT"); exist {
		Config.RedisPort = env
	} else {
		logger.Error("cannot find env REDIS_PORT")
		return errors.New("cannot find env REDIS_PORT")
	}

	if env, exist := os.LookupEnv("REDIS_PASSWORD"); exist {
		Config.RedisPassword = env
	} else {
		logger.Error("cannot find env REDIS_PASSWORD")
		return errors.New("cannot find env REDIS_PASSWORD")
	}

	if env, exist := os.LookupEnv("REDIS_DB"); exist {
		val, err := strconv.Atoi(env)
		if err != nil {
			logger.Error("cannot parse env REDIS_DB", "error", err)
			return errors.New("cannot parse env REDIS_DB")
		}
		Config.RedisDB = val
	} else {
		logger.Error("cannot find env REDIS_DB")
		return errors.New("cannot find env REDIS_DB")
	}

	// AI Provider configuration
	if env, exist := os.LookupEnv("AI_PROVIDER"); exist {
		Config.AIProvider = env
	} else {
		Config.AIProvider = "perplexity" // Default to Perplexity
	}

	// Perplexity API Key (required if using Perplexity)
	if env, exist := os.LookupEnv("PERPLEXITY_API_KEY"); exist {
		Config.PerplexityAPIKey = env
	} else if Config.AIProvider == "perplexity" {
		logger.Error("cannot find env PERPLEXITY_API_KEY (required for Perplexity provider)")
		return errors.New("cannot find env PERPLEXITY_API_KEY")
	}

	// OpenAI API Key (now conditional - only required if using OpenAI)
	if env, exist := os.LookupEnv("OPENAI_API_KEY"); exist {
		Config.OpenAIAPIKey = env
	} else if Config.AIProvider == "openai" {
		logger.Error("cannot find env OPENAI_API_KEY (required for OpenAI provider)")
		return errors.New("cannot find env OPENAI_API_KEY")
	}

	if env, exist := os.LookupEnv("S3_ENDPOINT"); exist {
		Config.S3Endpoint = env
	} else {
		logger.Error("cannot find env S3_ENDPOINT")
		return errors.New("cannot find env S3_ENDPOINT")
	}

	if env, exist := os.LookupEnv("S3_PUBLIC_BASE_URL"); exist {
		Config.S3PublicBaseURL = env
	} else {
		Config.S3PublicBaseURL = "https://s3.nutri02.com"
	}

	if env, exist := os.LookupEnv("S3_ACCESS_KEY"); exist {
		Config.S3AccessKey = env
	} else {
		logger.Error("cannot find env S3_ACCESS_KEY")
		return errors.New("cannot find env S3_ACCESS_KEY")
	}

	if env, exist := os.LookupEnv("S3_SECRET_KEY"); exist {
		Config.S3SecretKey = env
	} else {
		logger.Error("cannot find env S3_SECRET_KEY")
		return errors.New("cannot find env S3_SECRET_KEY")
	}

	if env, exist := os.LookupEnv("S3_BUCKET_NAME"); exist {
		Config.S3BucketName = env
	} else {
		logger.Error("cannot find env S3_BUCKET_NAME")
		return errors.New("cannot find env S3_BUCKET_NAME")
	}

	if env, exist := os.LookupEnv("S3_USE_SSL"); exist {
		val, err := strconv.ParseBool(env)
		if err != nil {
			logger.Error("cannot parse env S3_USE_SSL", "error", err)
			return errors.New("cannot parse env S3_USE_SSL")
		}
		Config.S3UseSSL = val
	} else {
		Config.S3UseSSL = true
	}

	if env, exist := os.LookupEnv("S3_REGION"); exist {
		Config.S3Region = env
	} else {
		Config.S3Region = "us-east-1"
	}

	// Service token for internal communication (optional, but required for /internal endpoints)
	if env, exist := os.LookupEnv("SERVICE_TOKEN"); exist {
		Config.ServiceToken = env
	} else {
		logger.Warn("SERVICE_TOKEN not set, internal endpoints will be disabled")
	}

	return nil
}
