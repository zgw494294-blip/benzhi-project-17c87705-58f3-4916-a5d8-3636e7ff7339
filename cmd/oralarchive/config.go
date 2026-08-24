package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultAddr = "127.0.0.1:19081"

type config struct {
	addr      string
	dataDir   string
	selfcheck bool
}

func parseConfig() (config, error) {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", defaultAddr, "HTTP 监听地址")
	flag.StringVar(&cfg.dataDir, "data", ".oralarchive-data", "本地数据目录")
	flag.BoolVar(&cfg.selfcheck, "selfcheck", false, "运行完整 HTTP 自检后退出")
	flag.Parse()
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" && cfg.addr == defaultAddr {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return cfg, fmt.Errorf("PORT 必须是 1-65535 的端口号")
		}
		cfg.addr = net.JoinHostPort("127.0.0.1", port)
	}
	host, port, err := net.SplitHostPort(cfg.addr)
	if err != nil {
		return cfg, fmt.Errorf("-addr 无效: %w", err)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return cfg, fmt.Errorf("-addr 必须绑定回环地址")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return cfg, fmt.Errorf("-addr 端口无效")
	}
	return cfg, nil
}
