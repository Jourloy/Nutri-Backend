package lib

import (
	"errors"
	"os"

	"github.com/charmbracelet/log"
)

type JwtData struct {
	Id string `json:"id"`
}

type conf struct {
	DatabaseDSN           string // DSN link for connect with database
	JWTSecret             string
	TbankTerminalKey      string
	TbankTerminalPassword string
	TelegramToken         string // Optional: used to proxy Telegram avatars
	MyURL                 string
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

	if env, exist := os.LookupEnv("MY_URL"); exist {
		Config.MyURL = env
	} else {
		logger.Error("cannot find env MY_URL")
		return errors.New("cannot find env MY_URL")
	}

	if env, exist := os.LookupEnv("TELEGRAM_TOKEN"); exist {
		Config.TelegramToken = env
	} else {
		logger.Error("cannot find env TELEGRAM_TOKEN")
		return errors.New("cannot find env TELEGRAM_TOKEN")
	}

	return nil
}
