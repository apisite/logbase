package logbase

import (
	"compress/bzip2"
	"compress/flate"
	"compress/gzip"
	"context"
	"io"
	//"github.com/google/uuid"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/jackc/pgx/v4/pgxpool"
	"gopkg.in/birkirb/loggers.v1"

	"github.com/apisite/logbase/nginx"
)

type Config struct {
	FileBeginProc string `long:"file_begin" default:"logs.file_begin" description:"Before file load func"`
}

// FileConfig holds config loaded from DB via key
type FileConfig struct {
	ID   int
	Type LogType
	Data []byte
}

// Service holds service data
type Service struct {
	Config     *Config
	Log        loggers.Contextual
	DB         *pgxpool.Pool
	commitLock sync.RWMutex
	//	MessageChan chan interface{}
}

// New creates an LogBase object
func New(cfg Config, log loggers.Contextual, db *pgxpool.Pool) *Service {
	srv := &Service{
		Config: &cfg,
		Log:    log,
		DB:     db,
		//	MessageChan: make(chan interface{}),
	}
	return srv
}

func (srv *Service) Auth(key string) (*FileConfig, error) {

	ctx := context.Background()
	db, err := srv.DB.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Release()

	fc := FileConfig{}
	sql := "select id, type_id, data from logs.config where key=$1"
	if err := db.QueryRow(ctx, sql, key).Scan(&fc.ID, &fc.Type, &fc.Data); err != nil {
		return nil, err
	}
	srv.Log.Printf("Got key for: %s/%d", fc.Type, fc.ID)
	return &fc, nil
}

func (srv *Service) LoadFile(cfg *FileConfig, path, file, ctype string) (int, error) {

	ctx := context.Background()
	var fileID int
	db, err := srv.DB.Acquire(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "Acquire")
	}
	defer db.Release()

	sql := "select logs.file_before(a_type_id => $1, a_config_id => $2, a_filename => $3)"
	if err := db.QueryRow(ctx, sql, cfg.Type, cfg.ID, file).Scan(&fileID); err != nil {
		return 0, errors.Wrap(err, "FileBegin")
	}
	filePath := filepath.Join(path, file)
	go srv.load(cfg, filePath, ctype, fileID)
	return fileID, nil
}

func (srv *Service) load(cfg *FileConfig, file, ctype string, fileID int) {
	// file_begin
	ctx := context.Background()
	fh, err := os.Open(file)
	if err != nil {
		srv.Log.Errorf("Open %s error: %v", file, err)
		return
	}
	defer fh.Close()
	srv.Log.Warnf("File %s encoding: %s", file, ctype)

	var reader io.Reader
	switch ctype {
	case "bz2":
		reader = bzip2.NewReader(fh)
	case "gzip":
		gz, err := gzip.NewReader(fh)
		if err != nil {
			srv.Log.Errorf("Open gzip %s error: %v", file, err)
			return
		}
		reader = gz
	case "deflate":
		def := flate.NewReader(fh)
		reader = def
	default:
		// just use the default reader
		reader = fh
	}
	//defer reader.Close()

	var stat []interface{}
	switch cfg.Type {
	case Nginx:
		srv.Log.Print("Load nginx: " + file)
		nstat := nginx.Stat{}
		err = nginx.Run(srv.DB, cfg.Data, fileID, reader, &nstat)
		stat = []interface{}{fileID, err, nstat.Total, nstat.Loaded, nstat.Skipped, nstat.First, nstat.Last}
	default:
		err = errors.New("Unknown config type")
	}

	// file_end
	db, err := srv.DB.Acquire(ctx)
	if err != nil {
		srv.Log.Errorf("Acquire: %v", err)
		return
	}
	defer db.Release()
	sql := "select logs.file_after(a_id => $1, a_error => $2, a_total=>$3, a_loaded=>$4, a_skipped=>$5, a_first => $6, a_last => $7)"
	if _, err := db.Exec(ctx, sql, stat...); err != nil {
		srv.Log.Errorf("FileEnd: %v", err)
	}
	srv.Log.Printf("File stat: %+v", stat)
}
