package main

import "testing"

func TestParseConfigRejectsNonLoopback(t *testing.T) {
	if _, err := parseConfig([]string{"-addr=0.0.0.0:19081"}); err == nil {
		t.Fatal("不应允许非回环监听")
	}
	cfg, err := parseConfig([]string{"-addr=127.0.0.1:19123", "-data=testdata"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.addr != "127.0.0.1:19123" || cfg.dataDir != "testdata" {
		t.Fatalf("配置解析异常: %+v", cfg)
	}
}
