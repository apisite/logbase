package nginx

import (
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
	Channels int    `json:"channels"`
	Skip     string `json:"skip"` // SkipRegExp
	Host     string `json:"host"` // HostRegExp
	Format   string `json:"format"`
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
	// Create reader and call Read method until EOF
	reader := gonx.NewReader(source, config.Format)

	rowsChan := make(chan map[string]interface{})
	loadedChan := make(chan int, config.Channels)     // get counts of loaded rows
	rowNumChan := make(chan int, config.Channels*2+1) // get 0 and first and last loaded row
	wg := sync.WaitGroup{}
	wg.Add(config.Channels)
	for i := 0; i < config.Channels; i++ {
		go stream(db, &wg, rowsChan, loadedChan, rowNumChan)
	}
	var reSkip, reHost *regexp.Regexp
	if reSkip, err = regexp.Compile(config.Skip); err != nil {
		return
	}
	if reHost, err = regexp.Compile(config.Host); err != nil {
		return
	}
	var stampID int
	//defer elapsed("File")()
	for {
		rec, er := reader.Read()
		if er == io.EOF {
			fmt.Println("EOF")
			break
		} else if er != nil {
			err = er
			return
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
					row["url"], args, err = parseURL(fs[1])
					if err != nil {
						return err
					}
					var a []byte
					a, err = json.Marshal(args)
					if err != nil {
						return err
					}
					row["args"] = string(a) // []string{string(a)}
				} else {
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
		row["id"] = stampID
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
				row["ref_url"], args, err = parseURL(ref)
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
		rowsChan <- row
	}
	fmt.Printf("S1a")
	close(rowsChan)
	fmt.Printf("S1b")
	wg.Wait()
	fmt.Printf("S1c")
	close(loadedChan)
	close(rowNumChan)
	for l := range loadedChan {
		stat.Loaded += l
	}
	fmt.Printf("S1d")
	for i := range rowNumChan {
		if i > stat.Last {
			stat.Last = i
		}
		if i < stat.First {
			stat.First = i
		}
	}

	fmt.Printf("S2")
	return
}

func stream(pool *pgxpool.Pool, wg *sync.WaitGroup, rowChan chan (map[string]interface{}), loadedChan chan int, rowNumChan chan int) {
	defer wg.Done()
	ctx := context.Background()
	db, err := pool.Acquire(ctx)
	if err != nil {
		panic(err)
	}

	defer db.Release()
	sql := "select logs.request_add(a_stamp =>$1, a_addr=>$2, a_url=>$3, a_referer=>$4, a_agent =>$5, a_method =>$6, a_status => $7,a_size =>$8,a_fresp =>$9, a_fload =>$10, a_args => $11, a_ref_url => $12, a_ref_args => $13, a_stamp_id => $14, a_row_num => $15)"
	fields := []string{"time_local", "remote_addr", "url", "referer", "user_agent", "method", "status", "size", "fresp", "fload", "args", "ref_url", "ref_args", "id", "line_num"}

	var load, total, rowNum int
	defer func() {
		fmt.Printf("Rows: %d Loaded: %d Last: %d\n", total, load, rowNum)
	}()
	for row := range rowChan {
		var found bool
		vals := values(row, fields...)
		err = db.QueryRow(ctx, sql, vals...).Scan(&found)
		if err != nil {
			fmt.Printf("\nLoad for %+v error %+v\n", row, err)
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

		if total%1000 == 0 {
			fmt.Printf(".") // TODO: update stat
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
func parseURL(s string) (u string, args map[string][]string, err error) {
	u1, err := url.Parse(s)
	if err != nil {
		fmt.Printf("\nURL %v\nparse error: %v\n", s, err)
		return
	}
	a := u1.Query()

	if len(a) > 0 {
		raw := u1.RawQuery //fmt.Sprintf("%v", q) // TODO: make json with cp1251
		if !strings.HasPrefix(u1.Path, "/api/") {
			rv := map[string][]string{}
			// args in 1251
			for k, v := range a {
				rv[k] = []string{}
				for _, i := range v {
					sr := strings.NewReader(i)
					tr := transform.NewReader(sr, charmap.Windows1251.NewDecoder())
					buf, er := ioutil.ReadAll(tr)
					if er != nil {
						fmt.Printf("%+v error: %v\n", raw, err)
						err = er
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
