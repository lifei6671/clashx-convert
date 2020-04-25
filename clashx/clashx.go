package clashx

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"strings"
	"sync"
)

var (
	VmessPrefix = []byte("vmess://")
	converter   = make(map[string]Converter)
	lock        = &sync.RWMutex{}
)

type Converter interface {
	Convert(body string) (*Config, error)
}

type Config struct {
	//HTTP 代理端口
	Port int `yaml:"port"`
	//SOCKS5 代理端口
	SocksPort int `yaml:"socks-port"`
	//允许局域网的连接（可用来共享代理）
	AllowLan bool `yaml:"allow-lan"`
	//此功能仅在 allow-lan 设置为 true 时生效
	BindAddress string `yaml:"bind-address"`
	//规则模式：Rule（规则） / Global（全局代理）/ Direct（全局直连）
	Mode string `yaml:"mode"`
	// 设置日志输出级别 (默认级别：silent，即不输出任何内容，以避免因日志内容过大而导致程序内存溢出）。
	// 5 个级别：silent / info / warning / error / debug。级别越高日志输出量越大，越倾向于调试，若需要请自行开启。
	LogLevel string `yaml:"log-level"`
	// clash 的 RESTful API
	ExternalController string `yaml:"external-controller"`
	// 您可以将静态网页资源（如 clash-dashboard）放置在一个目录中，clash 将会服务于 `${API}/ui`
	// 参数应填写配置目录的相对路径或绝对路径。
	ExternalUi string `yaml:"external-ui"`
	//RESTful API 的口令 (可选)
	Secret     string        `yaml:"secret"`
	Proxy      []*Proxy      `yaml:"Proxy"`
	ProxyGroup []*ProxyGroup `yaml:"Proxy Group"`
	Rule       []string      `yaml:"Rule"`
}

func (m *Config) String() string {
	if m == nil {
		return ""
	}

	b, err := yaml.Marshal(m)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(b)
}

type Proxy struct {
	Name           string            `yaml:"name"`
	Type           string            `yaml:"type"`
	Server         string            `yaml:"server"`
	Port           int               `yaml:"port"`
	UUID           string            `yaml:"uuid"`
	AlterId        int               `yaml:"alterId"`
	UDP            bool              `yaml:"udp"`
	SkipCertVerify bool              `yaml:"skip-cert-verify"`
	WSPath         string            `yaml:"ws-path"`
	WSHeaders      map[string]string `yaml:"ws-headers"`
	Cipher         string            `yaml:"cipher"`
	TLS            bool              `yaml:"tls"`
	Password       string            `yaml:"password"`
	Plugin         string            `yaml:"plugin"`
	PluginOpts     map[string]string `yaml:"plugin-opts"`
	Network        string            `yaml:"network"`
}

func (m *Proxy) String() string {
	b, err := yaml.Marshal(m)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(b)
}

type ProxyGroup struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Proxies  []string `yaml:"proxies"`
	Url      string   `yaml:"url"`
	Interval int      `yaml:"interval"`
}

type VmessClashX struct {
	c *Config
}

func NewVmessClashX(config io.Reader) *VmessClashX {
	c := &Config{}
	if err := yaml.NewDecoder(config).Decode(c); err != nil {
		log.Panicln(err)
	}
	if len(c.ProxyGroup) == 0 {
		c.ProxyGroup = make([]*ProxyGroup, 1)
		c.ProxyGroup[0] = &ProxyGroup{
			Name:     "Proxy",
			Type:     "select",
			Proxies:  make([]string, 0),
			Url:      "",
			Interval: 0,
		}
	}
	return &VmessClashX{c: c}
}

func (m *VmessClashX) Convert(body string) (*Config, error) {
	body = strings.ReplaceAll(body, " ", "")
	b, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, err
	}

	for _, bb := range bytes.Split(b, []byte("\n")) {
		proxy := &Proxy{}
		if bytes.HasPrefix(bb, VmessPrefix) {
			proxy.Type = "vmess"
			bbb, err := base64.StdEncoding.DecodeString(string(bytes.TrimPrefix(bb, VmessPrefix)))
			if err != nil {
				return nil, err
			}
			data := V2rayConfig{}

			if err := json.Unmarshal(bbb, &data); err != nil {
				log.Println(err)
				return nil, err
			}
			proxy.Name = data.Ps
			proxy.Server = data.Add
			if port, err := data.Port.Int64(); err == nil {
				proxy.Port = int(port)
			}
			proxy.UUID = data.Id
			if aid, err := data.Aid.Int64(); err == nil {
				proxy.AlterId = int(aid)
			}
			proxy.Cipher = data.Type
			proxy.TLS = data.TLS == "tls"
			if data.Net == "ws" {
				proxy.Network = data.Net
				proxy.WSPath = data.Path
				proxy.WSHeaders = map[string]string{"Host": data.Host}
			}

			m.c.Proxy = append(m.c.Proxy, proxy)
			for _, g := range m.c.ProxyGroup {
				g.Proxies = append(g.Proxies, proxy.Name)
			}
		}
	}

	return m.c, nil
}

func SingleVmessConvert(body string) (*Proxy, error) {
	bbb, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(body, string(VmessPrefix)))
	if err != nil {
		return nil, err
	}
	data := V2rayConfig{}
	if err := json.Unmarshal(bbb, &data); err != nil {
		log.Println(err)
		return nil, err
	}
	proxy := &Proxy{}
	proxy.Name = data.Ps
	proxy.Server = data.Add
	proxy.Type = "vmess"
	if port, err := data.Port.Int64(); err == nil {
		proxy.Port = int(port)
	}
	proxy.UUID = data.Id

	if aid, err := data.Aid.Int64(); err == nil {
		proxy.AlterId = int(aid)
	}
	proxy.Cipher = data.Type
	proxy.TLS = data.TLS == "tls"
	if data.Net == "ws" {
		proxy.Network = data.Net
		proxy.WSPath = data.Path
		proxy.WSHeaders = map[string]string{"Host": data.Host}
	}
	return proxy, nil
}

func Register(name string, c Converter) {
	lock.Lock()
	defer lock.Unlock()
	converter[name] = c
}

func GetConverter(name string) Converter {
	lock.RLock()
	defer lock.RUnlock()
	if c, ok := converter[name]; ok {
		return c
	}
	return nil
}

func init() {
	Register("vmess", NewVmessClashX(strings.NewReader(ConfigStr)))
}
