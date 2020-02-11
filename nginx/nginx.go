package nginx

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/satyrius/gonx"
)

type Config struct {
	Channels   int      `json:"channels"`
	Skip       string   `json:"skip"` // SkipRegExp
	Host       string   `json:"host"` // HostRegExp
	Format     string   `json:"format"`
	Fields     []string `json:"fields"`
	UTF8Prefix string   `json:"utf8_prefix"` // url with args encoded in utf8 (cp1251 used otherwise)
}

type Stat struct {
	Total   int
	Skipped int
	Loaded  int
	First   int
	Last    int
}

func Run(db *pgxpool.Pool, conf []byte, fileID int, source io.Reader, stat *Stat) (err error) {
	config := Config{}
	if err = json.Unmarshal([]byte(conf), &config); err != nil {
		return
	}
	if config.UTF8Prefix == "" {
		config.UTF8Prefix = "/"
	}
	// Create reader and call Read method until EOF
	// gonx.Reader uses goroutines and row order not fixed, so we use just parser
	// reader := gonx.NewReader(source, config.Format)
	parser := gonx.NewParser(config.Format)

	rowsChan := make(chan map[string]interface{})
	loadedChan := make(chan int, config.Channels)     // get count of loaded rows
	rowNumChan := make(chan int, config.Channels*2+1) // get 0, first and last loaded row
	wg := sync.WaitGroup{}
	wg.Add(config.Channels)
	for i := 0; i < config.Channels; i++ {
		go stream(db, i, &wg, rowsChan, loadedChan, rowNumChan, config.Fields)
	}
	var reSkip, reHost *regexp.Regexp
	if reSkip, err = regexp.Compile(config.Skip); err != nil {
		return
	}
	if reHost, err = regexp.Compile(config.Host); err != nil {
		return
	}

	defer func() {
		close(rowsChan)
		wg.Wait()
		close(loadedChan)
		close(rowNumChan)
		for l := range loadedChan {
			stat.Loaded += l
		}
		for i := range rowNumChan {
			if i > stat.Last {
				stat.Last = i
			}
			if i < stat.First {
				stat.First = i
			}
		}
	}()
	var stampID int
	//defer elapsed("File")()

	reader := bufio.NewReader(source)
	for {
		line, er := reader.ReadString('\n')
		if er == io.EOF {
			break
		} else if er != nil {
			err = er
			return
		}
		// Parse record
		rec, er := parser.ParseString(line)
		if er != nil {
			fmt.Printf("Rec error: %+v\n", er)
			continue
		}
		// Process the record... e.g.
		row := map[string]interface{}{}
		var skip bool
		for v := range rec.Fields() {
			s, err := rec.Field(v)
			if err != nil {
				return err
			}
			if v == "request" {
				fs := strings.Split(s, " ")
				if len(fs) == 3 {
					row["method"] = fs[0]
					row["proto"] = fs[2]
					var args map[string][]string
					row["url"], args, err = parseURL(fs[1], config.UTF8Prefix)
					if err != nil {
						fmt.Printf("\nURL parse error: %v\n", err)
						row["url"] = fs[1]
					} else {
						var a []byte
						a, err = json.Marshal(args)
						if err != nil {
							fmt.Printf("\nArgs marshal error: %v\n", err)
						} else {
							row["args"] = string(a)
						}
					}
				} else {
					// not HTTP request
					skip = true
				}
			} else {
				row[v] = s
			}
		}
		if stat.Total == 0 {
			// first file's timestamp
			stamp, ok := row["time_local"]
			if !ok {
				err = errors.New("required field 'time_local' not found")
				return
			}
			stampID, err = registerStamp(db, fileID, stamp)
			fmt.Printf("File stamp: %v\n", stamp)
			if err != nil {
				return
			}
		}
		row["file_id"] = fileID
		row["stamp_id"] = stampID
		row["line_num"] = stat.Total
		stat.Total++

		if skip {
			fmt.Printf("\nSkip: %+v\n", row)
			continue
		}

		if reSkip.MatchString(row["url"].(string)) {
			stat.Skipped++
			continue
		}
		if ref, ok := row["referer"].(string); ok {
			if reHost.MatchString(ref) {
				// internal uri
				var args map[string][]string
				row["ref_url"], args, err = parseURL(ref, config.UTF8Prefix)
				if err != nil {
					return err
				}
				var a []byte
				a, err = json.Marshal(args)
				if err != nil {
					return err
				}
				row["ref_args"] = string(a)
				delete(row, "referer")
			}
		}
		//fmt.Printf("Rec: %+v\n", row)
		rowsChan <- row
	}
	return
}

func stream(pool *pgxpool.Pool, id int, wg *sync.WaitGroup, rowChan chan (map[string]interface{}), loadedChan chan int, rowNumChan chan int, fields []string) {
	defer wg.Done()
	ctx := context.Background()
	db, err := pool.Acquire(ctx)
	if err != nil {
		panic(err)
	}
	defer db.Release()

	placeHolders := []string{}
	for i, n := range fields {
		placeHolders = append(placeHolders, fmt.Sprintf("a_%s => $%d", n, i+1))
	}

	sql := fmt.Sprintf("select logs.request_add(%s)", strings.Join(placeHolders, ","))
	updateSQL := "select logs.file_update_stat(a_id => $1, a_total => $2, a_loaded => $3)"

	var load, total, rowNum int
	defer func() {
		fmt.Printf("Process %d: Rows: %d Loaded: %d Last: %d\n", id, total, load, rowNum)
	}()
	updateOffset := id * 200
	var sentOffset, loadOffset int
	for row := range rowChan {
		var found bool
		vals := values(row, fields...)
		err = db.QueryRow(ctx, sql, vals...).Scan(&found)
		if err != nil {
			fmt.Printf("Load for %+v error %+v\n", row, err)
		}
		total++
		if found {
			load++
			if rowNum == 0 {
				// first loaded row
				rowNumChan <- row["line_num"].(int)
			}
			rowNum = row["line_num"].(int) // store as last loaded row
		}

		if (total+updateOffset)%1000 == 0 {
			_, er := db.Exec(ctx, updateSQL, row["file_id"], total-sentOffset, load-loadOffset)
			sentOffset = total
			loadOffset = load
			if er != nil {
				fmt.Printf("Update stat error %+v\n", er)
			}
		}
	}
	loadedChan <- load
	rowNumChan <- rowNum
}

func registerStamp(pool *pgxpool.Pool, fileID int, stamp interface{}) (int, error) {

	ctx := context.Background()
	db, err := pool.Acquire(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "Acquire")
	}
	defer db.Release()

	var stampID int
	sql := "select logs.stamp_register(a_file_id => $1, a_stamp => $2)"
	if err := db.QueryRow(ctx, sql, fileID, stamp).Scan(&stampID); err != nil {
		return 0, errors.Wrap(err, "StampRegister")
	}
	return stampID, nil
}

// parseURL returns page url as string and GET args as map
func parseURL(s, UTF8Prefix string) (u string, args map[string][]string, err error) {
	u1, err := url.Parse(s)
	if err != nil {
		return
	}
	a := u1.Query()

	if len(a) > 0 {
		raw := u1.RawQuery
		if !strings.HasPrefix(u1.Path, UTF8Prefix) {
			rv := map[string][]string{}
			// args in 1251
			for k, v := range a {
				rv[k] = []string{}
				for _, i := range v {
					sr := strings.NewReader(i)
					tr := transform.NewReader(sr, charmap.Windows1251.NewDecoder())
					buf, er := ioutil.ReadAll(tr)
					if er != nil {
						fmt.Printf("%+v decode error: %v\n", raw, err)
						err = errors.Wrap(er, "args decode")
						return
					}
					rv[k] = append(rv[k], string(buf))
				}
			}
			args = rv
		} else {
			args = a
		}
	}
	u = u1.Path
	return
}

// values fills a slice with map values in given names order
func values(data map[string]interface{}, names ...string) []interface{} {
	rv := []interface{}{}
	var null *int
	for _, n := range names {
		d, ok := data[n]
		if !ok || (n == "fload" && d == "-") {
			rv = append(rv, null)
		} else {
			rv = append(rv, d)
		}
	}
	return rv
}

func elapsed(id int) func() {
	start := time.Now()
	return func() {
		fmt.Printf("File %d took %v\n", id, time.Since(start))
	}
}
