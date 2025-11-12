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
	OpenAIAPIKey    string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucketName string
	MinioUseSSL     bool
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
		logger.Error("cannot find env MY_URL")
		return errors.New("cannot find env MY_URL")
	}

	if env, exist := os.LookupEnv("FRONT_URL"); exist {
		Config.FrontURL = env
	} else {
		logger.Error("cannot find env FRONT_URL")
		return errors.New("cannot find env FRONT_URL")
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

	// AI and Storage configuration
	if env, exist := os.LookupEnv("OPENAI_API_KEY"); exist {
		Config.OpenAIAPIKey = env
	} else {
		logger.Error("cannot find env OPENAI_API_KEY")
		return errors.New("cannot find env OPENAI_API_KEY")
	}

	if env, exist := os.LookupEnv("MINIO_ENDPOINT"); exist {
		Config.MinioEndpoint = env
	} else {
		logger.Error("cannot find env MINIO_ENDPOINT")
		return errors.New("cannot find env MINIO_ENDPOINT")
	}

	if env, exist := os.LookupEnv("MINIO_ACCESS_KEY"); exist {
		Config.MinioAccessKey = env
	} else {
		logger.Error("cannot find env MINIO_ACCESS_KEY")
		return errors.New("cannot find env MINIO_ACCESS_KEY")
	}

	if env, exist := os.LookupEnv("MINIO_SECRET_KEY"); exist {
		Config.MinioSecretKey = env
	} else {
		logger.Error("cannot find env MINIO_SECRET_KEY")
		return errors.New("cannot find env MINIO_SECRET_KEY")
	}

	if env, exist := os.LookupEnv("MINIO_BUCKET_NAME"); exist {
		Config.MinioBucketName = env
	} else {
		Config.MinioBucketName = "nutri-ai-images" // Default bucket name
	}

	if env, exist := os.LookupEnv("MINIO_USE_SSL"); exist {
		val, err := strconv.ParseBool(env)
		if err != nil {
			logger.Error("cannot parse env MINIO_USE_SSL", "error", err)
			return errors.New("cannot parse env MINIO_USE_SSL")
		}
		Config.MinioUseSSL = val
	} else {
		Config.MinioUseSSL = true // Default to true for security
	}

	return nil
}
