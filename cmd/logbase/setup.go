package main

import (
	"context"
	"errors"

	"github.com/jessevdk/go-flags"

	mapper "github.com/birkirb/loggers-mapper-logrus"
	"github.com/sirupsen/logrus"
	"gopkg.in/birkirb/loggers.v1"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/apisite/logbase"
)

// Config holds all config vars
type Config struct {
	Listen    string         `long:"listen" default:":8080" description:"Addr and port which server listens at"`
	DSN       string         `long:"dsn" default:"" description:"Database URL"`
	MaxConns  int32          `long:"max_conns" default:"10" description:"DB max conns"`
	Path      string         `long:"path" default:"/tmp" description:"Path to store loaded files"`
	Verbose   bool           `long:"verbose" description:"Show debug data"`
	KeyHeader string         `long:"key_header" default:"Auth" description:"Logbase key header"`
	LogBase   logbase.Config `group:"LogBase Options" namespace:"lb"`
}

var (
	// ErrGotHelp returned after showing requested help
	ErrGotHelp = errors.New("help printed")
	// ErrBadArgs returned after showing command args error message
	ErrBadArgs = errors.New("option error printed")
)

// setupConfig loads flags from args (if given) or command flags and ENV otherwise
func setupConfig(args ...string) (*Config, error) {
	cfg := &Config{}
	p := flags.NewParser(cfg, flags.Default) //  HelpFlag | PrintErrors | PassDoubleDash
	var err error
	if len(args) == 0 {
		_, err = p.Parse()
	} else {
		_, err = p.ParseArgs(args)
	}
	if err != nil {
		//fmt.Printf("Args error: %#v", err)
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, ErrGotHelp
		}
		return nil, ErrBadArgs
	}
	return cfg, nil
}

// setupLog creates logger
func setupLog(cfg *Config) loggers.Contextual {
	l := logrus.New()
	if gin.IsDebugging() {
		l.SetLevel(logrus.DebugLevel)
		l.SetReportCaller(true)
	} else {
		l.SetLevel(logrus.WarnLevel)
	}
	return &mapper.Logger{Logger: l} // Same as mapper.NewLogger(l) but without info log message
}

func setupDB(cfg *Config) (pool *pgxpool.Pool, err error) {

	var config *pgxpool.Config
	config, err = pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return
	}
	//config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
	// do something with every new connection
	//}
	config.MaxConns = cfg.MaxConns
	pool, err = pgxpool.ConnectConfig(context.Background(), config)
	return
}
