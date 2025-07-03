package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// URLConfig 定义配置文件结构
type URLConfig struct {
	URLs     []URLItem `json:"urls"`
	Settings Settings  `json:"settings"`
}

// URLItem 定义单个 URL 的结构
type URLItem struct {
	URL string `json:"url"`
	IP  string `json:"ip"`
}

// Settings 定义全局设置的结构
type Settings struct {
	UpdateFreq int `json:"update_freq"`
	Timeout    int `json:"timeout"`
}

// AuthConfig 定义认证配置文件结构
type AuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	// 定义配置文件路径和监听端口
	configPath := flag.String("config", "url.json", "配置文件路径")
	authPath := flag.String("auth", "auth.json", "认证配置文件路径")
	listenAddr := flag.String("addr", ":9119", "监听地址和端口")
	flag.Parse()

	// 读取 URL 配置文件
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败：%v", err)
	}

	// 读取认证配置文件
	authConfig, err := loadAuthConfig(*authPath)
	if err != nil {
		log.Fatalf("加载认证配置文件失败：%v", err)
	}

	// 初始化 Prometheus 指标
	httpStatusCode := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_status_code",
			Help: "HTTP 状态码",
		},
		[]string{"url", "ip"},
	)

	// 注册指标
	prometheus.MustRegister(httpStatusCode)

	// 定期更新指标
	go func() {
		for {
			for _, item := range config.URLs {
				statusCode, err := checkURL(item.URL, config.Settings.Timeout)
				if err != nil {
					log.Printf("检测 URL %s 失败：%v 返回码： 000", item.URL, err)
					httpStatusCode.WithLabelValues(item.URL, item.IP).Set(0)
					continue
				}

				// 更新指标
				httpStatusCode.WithLabelValues(item.URL, item.IP).Set(float64(statusCode))
			}

			// 每隔指定的时间间隔更新一次指标
			time.Sleep(time.Duration(config.Settings.UpdateFreq) * time.Second)
		}
	}()

	// 创建一个自定义的指标过滤器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()
		registry.MustRegister(httpStatusCode)

		// 使用 promhttp.HandlerFor 来处理过滤后的指标
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})

	// 创建一个带 Basic Auth 的 /metrics 路径
	http.Handle("/metrics", basicAuthMiddleware(authConfig.Username, authConfig.Password, handler))

	// 启动 HTTP 服务
	log.Printf("开始监听 %s，更新频率为每 %d 秒，超时时间为 %d 秒", *listenAddr, config.Settings.UpdateFreq, config.Settings.Timeout)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("启动 HTTP 服务失败：%v", err)
	}
}

func loadConfig(path string) (*URLConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config URLConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func loadAuthConfig(path string) (*AuthConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var authConfig AuthConfig
	err = json.Unmarshal(data, &authConfig)
	if err != nil {
		return nil, err
	}

	return &authConfig, nil
}

func checkURL(url string, timeout int) (int, error) {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func basicAuthMiddleware(username, password string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
