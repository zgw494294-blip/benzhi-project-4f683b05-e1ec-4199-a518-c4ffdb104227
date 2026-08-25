package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type config struct {
	addr      string
	dataDir   string
	selfcheck bool
}

func parseConfig(args []string) (config, error) {
	defaultAddr := "127.0.0.1:19081"
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1024 || n > 65535 {
			return config{}, fmt.Errorf("PORT必须是1024到65535之间的端口号")
		}
		defaultAddr = net.JoinHostPort("127.0.0.1", port)
	}
	fs := flag.NewFlagSet("seedvault", flag.ContinueOnError)
	addr := fs.String("addr", defaultAddr, "仅回环监听地址")
	dataDir := fs.String("data", filepath.Join("data", "seedvault"), "数据目录")
	selfcheck := fs.Bool("selfcheck", false, "执行有界完整流程自检")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() != 0 {
		return config{}, fmt.Errorf("存在无法识别的参数")
	}
	host, port, err := net.SplitHostPort(*addr)
	if err != nil {
		return config{}, fmt.Errorf("-addr格式无效: %w", err)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return config{}, fmt.Errorf("监听地址必须为回环地址")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1024 || n > 65535 {
		return config{}, fmt.Errorf("监听端口必须在1024到65535之间")
	}
	if strings.TrimSpace(*dataDir) == "" {
		return config{}, fmt.Errorf("数据目录不能为空")
	}
	return config{addr: *addr, dataDir: *dataDir, selfcheck: *selfcheck}, nil
}
