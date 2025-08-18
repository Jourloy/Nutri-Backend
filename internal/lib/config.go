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
	DatabaseDSN string // DSN link for connect with database
	JWTSecret   string
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
		return errors.New("cannot find env DATABASE_DSN")
	}

	if env, exist := os.LookupEnv("JWT_SECRET"); exist {
		Config.JWTSecret = env
	} else {
		return errors.New("cannot find env JWT_SECRET")
	}

	return nil
}
