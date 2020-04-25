package server

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lifei6671/clashx-convert/clashx"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

var cache = &sync.Map{}
var changeChan = make(chan struct{}, 1)

type httpCache struct {
	Name         string `yaml:"name" json:"name"`
	ConfigName   string `yaml:"config_name" json:"config_name"`
	VmessPathUrl string `yaml:"vmess-path-url" json:"vmess_path_url"`
	Interval     int    `yaml:"interval" json:"interval"`
	Converter    string `yaml:"converter" json:"converter"`
	config       *clashx.Config
	cancel       context.CancelFunc
}

func Run(ctx context.Context, addr string, path string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, templateStr)
	})
	mux.HandleFunc("/config", config)
	mux.HandleFunc("/single-proxy", singleProxy)
	mux.HandleFunc("/add-subscribe", addSubscribe)

	host, port, _ := net.SplitHostPort(addr)
	if host == "" {
		host = "127.0.0.1"
	}
	log.Printf("Starting  http server -> http://%s:%s\n", host, port)

	server := &http.Server{Addr: addr, Handler: mux}

	initialize(ctx, path)
	go func() {
		select {
		case <-ctx.Done():
			cache.Range(func(key, value interface{}) bool {
				if c, ok := value.(*httpCache); ok {
					c.cancel()
				}
				return true
			})
			cache = &sync.Map{}
			_ = server.Shutdown(context.Background())
		}
	}()

	return server.ListenAndServe()
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
			w.Header().Add("Content-Type", "application/octet-stream")
			w.Header().Add("Content-Disposition", "attachment; filename=\""+c.ConfigName+".yaml\"")
			_, _ = fmt.Fprint(w, c.config.String())
			return
		}
	} else if urlStr := r.FormValue("url"); urlStr != "" {
		converter := r.FormValue("converter")
		if converter == "" {
			converter = "vmess"
		}

		config, err := get(urlStr, converter)
		if err != nil {
			log.Printf("Failed to get remote configuration -> %s %s", urlStr, err)
			_, _ = fmt.Fprint(w, err)
			return
		}
		if err := AddVmess(name, converter, urlStr, 60, config); err != nil {
			_, _ = fmt.Fprint(w, err)
			return
		}
		name, _ := getVmessName(urlStr)

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", "attachment; filename=\""+name+".yaml\"")

		_, _ = fmt.Fprint(w, config.String())
		return
	}
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", "attachment; filename=\""+name+".yaml\"")
	_, _ = fmt.Fprint(w, clashx.ConfigStr)

}

func singleProxy(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprint(w, err)
		return
	}
	singleProxyBody := r.FormValue("single_proxy")
	if singleProxyBody == "" {
		w.WriteHeader(400)
		_, _ = fmt.Fprint(w, "链接地址不能为空")
		return
	}
	config, err := clashx.SingleVmessConvert(singleProxyBody)
	if err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprint(w, err)
		return
	}
	w.Header().Add("Content-Disposition", "attachment; filename=\"config.yaml\"")

	_, _ = fmt.Fprint(w, config)
	return
}

func addSubscribe(w http.ResponseWriter, r *http.Request) {
	type subscribe struct {
		Port           json.Number `json:"port"`
		SocksPort      json.Number `json:"socks_port"`
		AllowLan       bool        `json:"allow_lan"`
		SubscribeInput string      `json:"subscribe_input"`
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(400)
		_, _ = fmt.Fprint(w, "解析请求数据失败")
		return
	}
	var model subscribe

	if err := json.Unmarshal(body, &model); err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprint(w, err)
		return
	}
	_, err = url.Parse(model.SubscribeInput)
	if err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprint(w, err)
		return
	}
	name, _ := getVmessName(model.SubscribeInput)
	if _, ok := cache.Load(name); ok {
		_, _ = fmt.Fprint(w, getDomain(r)+"/config?name="+name)
		return
	} else {
		converter := "vmess"

		config, err := get(model.SubscribeInput, converter)
		if err != nil {
			log.Printf("Failed to get remote configuration -> %s %s", model.SubscribeInput, err)
			w.WriteHeader(500)
			_, _ = fmt.Fprint(w, err)
			return
		}
		config.AllowLan = model.AllowLan
		if port, err := model.Port.Int64(); err == nil && port != 0 {
			config.Port = int(port)
		}
		if port, err := model.SocksPort.Int64(); err == nil && port != 0 {
			config.SocksPort = int(port)
		}
		if err := AddVmess(name, converter, model.SubscribeInput, 60, config); err != nil {
			w.WriteHeader(500)
			_, _ = fmt.Fprint(w, err)
			return
		}
		_, _ = fmt.Fprint(w, getDomain(r)+"/config?name="+name)
		return
	}
}

//AddVmess 增加一个配置转换.
func AddVmess(name, converter, urlStr string, interval int, oldConfig *clashx.Config) error {
	if c := clashx.GetConverter(converter); c == nil {
		return errors.New("Converter does not exist ->" + converter)
	}
	_, configName := getVmessName(urlStr)

	ctx, cancel := context.WithCancel(context.Background())
	hc := &httpCache{
		Name:         name,
		ConfigName:   configName,
		VmessPathUrl: urlStr,
		Interval:     interval,
		Converter:    converter,
		cancel:       cancel,
		config:       oldConfig,
	}

	actual, loaded := cache.LoadOrStore(name, hc)
	if loaded {
		actual.(*httpCache).cancel()
		cache.Store(name, hc)
	}
	log.Printf("增加配置成功 ->name=%s type=%s url=%s\n", name, converter, urlStr)
	go autoUpdateConfig(ctx, hc)

	changeChan <- struct{}{}
	return nil
}

func autoUpdateConfig(ctx context.Context, c *httpCache) {
	if c.Interval <= 0 {
		return
	}
	d := time.Minute * time.Duration(c.Interval)
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Automatic update has stopped ->cancel %s %s \n", c.Name, c.VmessPathUrl)
			return
		case <-timer.C:
			if _, ok := cache.Load(c.Name); !ok {
				log.Printf("Automatic update has stopped ->!ok %s %s \n", c.Name, c.VmessPathUrl)
				return
			}
			config, err := get(c.VmessPathUrl, c.Converter)
			if err != nil {
				log.Printf("Failed to get remote configuration -> %s %s", c.VmessPathUrl, err)
				break
			}
			log.Println("update completed ->", c.VmessPathUrl)
			oldConfig := c.config
			c.config = config
			if oldConfig != nil {
				c.config.AllowLan = oldConfig.AllowLan
				c.config.Port = oldConfig.Port
				c.config.SocksPort = oldConfig.SocksPort
			}
			timer.Reset(d)
		}
	}
}

func get(urlStr, converter string) (*clashx.Config, error) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(urlStr)

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

func getDomain(r *http.Request) string {
	port := fmt.Sprintf(":%d", getPort(r))
	if port == ":80" || port == ":443" || port == ":" {
		port = ""
	}
	scheme := getScheme(r)

	return fmt.Sprintf("%s://%s%s", scheme, getHost(r), port)
}

func getScheme(r *http.Request) string {
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	if r.URL.Scheme != "" {
		return r.URL.Scheme
	}
	if r.TLS == nil {
		return "http"
	}
	return "https"
}

func getHost(r *http.Request) string {
	if r.Host != "" {
		if hostPart, _, err := net.SplitHostPort(r.Host); err == nil {
			return hostPart
		}
		return r.Host
	}
	return "localhost"
}

func getPort(r *http.Request) int {
	if _, portPart, err := net.SplitHostPort(r.Host); err == nil {
		port, _ := strconv.Atoi(portPart)
		return port
	}
	return 80
}

func initialize(ctx context.Context, path string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-changeChan:
				var caches []*httpCache
				cache.Range(func(key, value interface{}) bool {
					if c, ok := value.(*httpCache); ok {
						caches = append(caches, c)
					}
					return true
				})
				if len(caches) > 0 {
					if f, err := os.Create(path); err == nil {
						encoder := gob.NewEncoder(f)
						if err := encoder.Encode(caches); err != nil {
							log.Println("备份失败 ->", err, path)
							break
						}
						log.Printf("备份成功 -> %s\n", path)
					}
				}
			}
		}
	}()
	if f, err := os.Open(path); err == nil {
		decoder := gob.NewDecoder(f)
		var caches []*httpCache
		err = decoder.Decode(&caches)
		if err != nil {
			log.Printf("解码失败 -> %s %s", err, path)
			return
		}
		for _, c := range caches {
			ctx1, cancel := context.WithCancel(ctx)
			tmp := *c
			tmp.cancel = cancel
			cache.Store(c.Name, &tmp)
			log.Printf("恢复备份成功 ->%s - %s\n", c.Name, path)
			go autoUpdateConfig(ctx1, c)
		}
	}
}

func getVmessName(urlStr string) (name string, configName string) {
	uri, err := url.Parse(urlStr)
	if err == nil {
		if uri.Path != "" {
			name := strings.TrimSuffix(filepath.Base(uri.Path), filepath.Ext(uri.Path))
			names := []rune(name)

			names[0] = unicode.ToUpper(names[0])

			configName = string(names)
		}
	}
	hash := md5.New()
	hash.Write([]byte(urlStr))
	name = hex.EncodeToString(hash.Sum(nil))
	if configName == "" {
		configName = name
	}
	return
}

func init() {
	gob.Register(&httpCache{})
}
