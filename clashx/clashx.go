package clashx

type Converter interface {
	Convert(body string) ([]*Config, error)
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
	Rule       []string      `yaml:"rule"`
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
	Password       string            `yaml:"password"`
	Plugin         string            `yaml:"plugin"`
	PluginOpts     map[string]string `yaml:"plugin-opts"`
}

type ProxyGroup struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Proxies  []string `yaml:"proxies"`
	Url      string   `yaml:"url"`
	Interval int      `yaml:"interval"`
}
