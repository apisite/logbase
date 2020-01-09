package logbase

import (
	"compress/bzip2"
	"context"
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

func (srv *Service) LoadFile(cfg *FileConfig, path, file string) (int, error) {

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
	go srv.load(cfg, filePath, fileID)
	return fileID, nil
}

func (srv *Service) load(cfg *FileConfig, file string, fileID int) {
	// file_begin
	ctx := context.Background()
	fh, err := os.Open(file)
	if err != nil {
		srv.Log.Errorf("Open %s error: %v", file, err)
		return
	}
	defer fh.Close()
	logReader := bzip2.NewReader(fh)

	var total, skip, load int
	switch cfg.Type {
	case Nginx:
		srv.Log.Print("Load nginx: " + file)
		total, skip, load, err = nginx.Run(srv.DB, cfg.Data, fileID, logReader)
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

	sql := "select logs.file_after(a_id => $1, a_error => $2, a_total=>$3, a_loaded=>$4, a_skipped=>$5)"
	if _, err := db.Exec(ctx, sql, fileID, err, total, load, skip); err != nil {
		srv.Log.Errorf("FileEnd: %v", err)
	}
	srv.Log.Printf("File %d load stat: total %d skip %d load %d", fileID, total, skip, load)
}
