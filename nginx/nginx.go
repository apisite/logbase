package nginx

import (
	"context"
	"encoding/json"
	"fmt"
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

func Run(db *pgxpool.Pool, conf []byte, fileID int, source io.Reader) (total int, skipped int, loaded int, err error) {
	config := Config{}
	if err = json.Unmarshal([]byte(conf), &config); err != nil {
		return
	}
	// Create reader and call Read method until EOF
	reader := gonx.NewReader(source, config.Format)

	rowsChan := make(chan map[string]interface{})
	loadedChan := make(chan int, config.Channels)
	wg := sync.WaitGroup{}
	for i := 0; i < config.Channels; i++ {
		wg.Add(1)
		go stream(db, &wg, rowsChan, loadedChan)
	}
	var reSkip, reHost *regexp.Regexp
	if reSkip, err = regexp.Compile(config.Skip); err != nil {
		return
	}
	if reHost, err = regexp.Compile(config.Host); err != nil {
		return
	}
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
		total++

		// Process the record... e.g.
		row := map[string]interface{}{
			"id": fileID,
		}
		var skip bool
		for v := range rec.Fields() {
			s, err := rec.Field(v)
			if err != nil {
				panic(err)
			}
			if v == "request" {
				fs := strings.Split(s, " ")
				if len(fs) == 3 {
					row["method"] = fs[0]
					row["proto"] = fs[2]
					var args map[string][]string
					row["url"], args, err = parseURL(fs[1])
					if err != nil {
						panic(err)
					}
					var a []byte
					a, err = json.Marshal(args)
					if err != nil {
						panic(err)
					}
					row["args"] = string(a) // []string{string(a)}

				} else {
					skip = true
				}
			} else {
				row[v] = s
			}
		}
		if skip {
			fmt.Printf("\nSkip: %+v\n", row)
			continue
		}

		if reSkip.MatchString(row["url"].(string)) {
			skipped++
			continue
		}
		if ref, ok := row["referer"].(string); ok {
			if reHost.MatchString(ref) {
				// internal uri
				var args map[string][]string
				row["ref_url"], args, err = parseURL(ref)
				if err != nil {
					panic(err)
				}
				var a []byte
				a, err = json.Marshal(args)
				if err != nil {
					panic(err)
				}
				row["ref_args"] = string(a)
				delete(row, "referer")
			}
		}
		rowsChan <- row
	}
	close(rowsChan)
	wg.Wait()
	close(loadedChan)
	for l := range loadedChan {
		loaded += l
	}
	return
}

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

func stream(pool *pgxpool.Pool, wg *sync.WaitGroup, rowChan chan (map[string]interface{}), loadedChan chan int) {
	ctx := context.Background()
	db, err := pool.Acquire(ctx)
	if err != nil {
		panic(err)
	}
	defer db.Release()
	sql := "select logs.request_add(a_stamp =>$1, a_addr=>$2, a_url=>$3, a_referer=>$4, a_agent =>$5, a_method =>$6, a_status => $7,a_size =>$8,a_fresp =>$9, a_fload =>$10, a_args => $11, a_ref_url => $12, a_ref_args => $13, a_file_id => $14)"
	fields := []string{"time_local", "remote_addr", "url", "referer", "user_agent", "method", "status", "size", "fresp", "fload", "args", "ref_url", "ref_args", "id"}

	var load, total int
	defer func() {
		fmt.Printf("Rows: %d Loaded: %d\n", total, load)
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
		}
		if total%1000 == 0 {
			fmt.Printf(".") // TODO: update stat
		}
	}
	loadedChan <- load
	wg.Done()
}

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
