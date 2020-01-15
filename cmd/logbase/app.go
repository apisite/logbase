package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	// TODO	"github.com/jackc/pgx/v4/log/logrusadapter"

	"github.com/apisite/logbase"
)

const (
	ContextLBConfig = "LB_CONFIG"
)

func run(exitFunc func(code int)) {
	var err error
	var cfg *Config
	defer func() { shutdown(exitFunc, err) }()
	cfg, err = setupConfig()
	if err != nil {
		return
	}
	l := setupLog(cfg)
	var db *pgxpool.Pool
	db, err = setupDB(cfg)
	if err != nil {
		return
	}
	lb := logbase.New(cfg.LogBase, l, db)

	router := gin.Default()
	authorized := router.Group("/upload")
	authorized.Use(AuthRequired(lb, cfg.KeyHeader))
	{
		authorized.POST("/nginx", UploadEndpoint(lb, logbase.Nginx, cfg.Path))
		//authorized.POST("/pg", pgUploadEndpoint)
		//authorized.POST("/journal", journalUploadEndpoint)
	}
	// TODO:  /upload/status?job - pub/sub
	router.Run(cfg.Listen)

}

// exit after deferred cleanups have run
func shutdown(exitFunc func(code int), e error) {
	if e != nil {
		var code int
		switch e {
		case ErrGotHelp:
			code = 3
		case ErrBadArgs:
			code = 2
		default:
			code = 1
			log.Printf("Run error: %+v", e)
		}
		exitFunc(code)
	}
}

// AuthRequired is a simple middleware to check the session
func AuthRequired(lb *logbase.Service, header string) func(c *gin.Context) {
	return func(c *gin.Context) {
		key := c.GetHeader(header)
		lbConf, err := lb.Auth(key)
		if err != nil {
			// Abort the request with the appropriate error code
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set(ContextLBConfig, lbConf)
		// Continue down the chain to handler etc
		c.Next()
	}
}

// login is a handler that parses a form and checks for specific data
func UploadEndpoint(lb *logbase.Service, logType logbase.LogType, path string) func(c *gin.Context) {
	return func(c *gin.Context) {

		lbConfIface, _ := c.Get(ContextLBConfig)
		if lbConfIface == nil {
			c.AbortWithError(500, errors.New("This endpoint must be under AuthRequired"))
		}
		lbConf := lbConfIface.(*logbase.FileConfig)

		if lbConf.Type != logType {
			c.AbortWithError(http.StatusUnauthorized, errors.New("This endpoint does not allower for given key"))
		}
		file := c.GetHeader("File")
		// TODO:
		// check for "",~/,~.
		// add path for logType
		ctype := c.GetHeader("Content-Encoding")
		body, err := c.GetRawData()
		if err != nil {
			c.AbortWithError(400, err)
		}
		r := bytes.NewReader(body)
		filePath := filepath.Join(path, file)
		fh, err := os.Create(filePath)
		if err != nil {
			c.AbortWithError(400, err)
		}

		n, err := io.Copy(fh, r)
		if err != nil {
			c.AbortWithError(400, err)
		}
		fh.Close()
		fileID, err := lb.LoadFile(lbConf, path, file, ctype)
		if err != nil {
			c.AbortWithError(400, err)
		}
		c.JSON(http.StatusOK, struct {
			ID    int
			Bytes int64
			File  string
			Type  string
		}{
			fileID,
			n,
			file,
			logType.String(),
		})
	}
}
