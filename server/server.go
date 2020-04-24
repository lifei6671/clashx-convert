package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/lifei6671/clashx-convert/clashx"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

var cache = &sync.Map{}

type httpCache struct {
	Name         string `yaml:"name" json:"name"`
	VmessPathUrl string `yaml:"vmess-path-url" json:"vmess_path_url"`
	Interval     int    `yaml:"interval" json:"interval"`
	Converter    string `yaml:"converter" json:"converter"`
	config       *clashx.Config
	cancel       context.CancelFunc
}

func Run(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "hello world!")
	})
	mux.HandleFunc("/config", config)

	log.Println("Starting  http server ->", addr)
	return http.ListenAndServe(addr, mux)
}

func config(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("Failed to parse parameters ->", err)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		_, _ = fmt.Fprint(w, "name is not null.")
		return
	}
	if content, ok := cache.Load(name); ok {
		if c, ok := content.(*httpCache); ok {
			if c.config == nil {
				config, err := get(c.VmessPathUrl, c.Converter)
				if err != nil {
					_, _ = fmt.Fprint(w, err)
					return
				}
				c.config = config
			}
			_, _ = fmt.Fprint(w, c.config.String())
			return
		}
	} else if urlStr := r.FormValue("url"); urlStr != "" {
		converter := r.FormValue("converter")
		if converter == "" {
			converter = "vmess"
		}
		if err := AddVmess(name, converter, urlStr, 60); err != nil {
			_, _ = fmt.Fprint(w, err)
			return
		}
		config, err := get(urlStr, converter)
		if err != nil {
			log.Printf("Failed to get remote configuration -> %s %s", urlStr, err)
			_, _ = fmt.Fprint(w, err)
			return
		}
		_, _ = fmt.Fprint(w, config.String())
		return
	}

	_, _ = fmt.Fprint(w, clashx.ConfigStr)

}

//AddVmess 增加一个配置转换.
func AddVmess(name, converter, urlStr string, interval int) error {
	if c := clashx.GetConverter(converter); c == nil {
		return errors.New("Converter does not exist ->" + converter)
	}
	ctx, cancel := context.WithCancel(context.Background())

	hc := &httpCache{
		Name:         name,
		VmessPathUrl: urlStr,
		Interval:     interval,
		Converter:    converter,
		cancel:       cancel,
	}

	actual, loaded := cache.LoadOrStore(name, hc)
	if loaded {
		actual.(*httpCache).cancel()
		cache.Store(name, hc)
	}
	log.Printf("增加配置成功 ->name=%s type=%s url=%s\n", name, converter, urlStr)
	go func() {
		if interval <= 0 {
			return
		}
		d := time.Minute * time.Duration(interval)
		timer := time.NewTimer(d)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Printf("Automatic update has stopped ->cancel %s %s \n", name, urlStr)
				return
			case <-timer.C:
				if _, ok := cache.Load(name); !ok {
					log.Printf("Automatic update has stopped ->!ok %s %s \n", name, urlStr)
					return
				}
				config, err := get(urlStr, converter)
				if err != nil {
					log.Printf("Failed to get remote configuration -> %s %s", urlStr, err)
					break
				}
				log.Println("update completed ->", urlStr)
				hc.config = config
				timer.Reset(d)

			}
		}
	}()
	return nil
}

func get(urlStr, converter string) (*clashx.Config, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		log.Println("Failed to get remote response ->", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read remote response -> %s %s", urlStr, err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to read remote response -> %s http_code=%d body=%s", urlStr, resp.StatusCode, string(body))
		return nil, fmt.Errorf("http_code=%d", resp.StatusCode)
	}
	config, err := clashx.GetConverter(converter).Convert(string(body))
	if err != nil {
		log.Println("Format conversion failed ->", err)
		return nil, err
	}
	return config, nil

}
