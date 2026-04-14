package config

import "testing"

func TestLoadShareHTTPConfig_defaults(t *testing.T) {
	t.Setenv(EnvShareHTTPHost, "")
	t.Setenv(EnvShareHTTPPort, "")
	t.Setenv(EnvShareHTTPBindAll, "")
	c, err := LoadShareHTTPConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.Host != "127.0.0.1" || c.Port != DefaultShareHTTPPort || c.BindAll {
		t.Fatalf("defaults: %#v", c)
	}
	if c.ListenHost() != "127.0.0.1" {
		t.Fatalf("ListenHost: %q", c.ListenHost())
	}
	if c.ClipboardBaseHost() != "127.0.0.1" {
		t.Fatalf("ClipboardBaseHost: %q", c.ClipboardBaseHost())
	}
}

func TestLoadShareHTTPConfig_bindAll(t *testing.T) {
	t.Setenv(EnvShareHTTPBindAll, "true")
	c, err := LoadShareHTTPConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !c.BindAll || c.ListenHost() != "0.0.0.0" {
		t.Fatalf("bind all: %#v listen=%q", c, c.ListenHost())
	}
	if c.ClipboardBaseHost() != "127.0.0.1" {
		t.Fatalf("clipboard host should stay loopback literal, got %q", c.ClipboardBaseHost())
	}
}

func TestLoadShareHTTPConfig_badPort(t *testing.T) {
	t.Setenv(EnvShareHTTPPort, "nope")
	if _, err := LoadShareHTTPConfig(); err == nil {
		t.Fatal("want error")
	}
}

func TestLoadShareHTTPConfig_ipv6Host(t *testing.T) {
	t.Setenv(EnvShareHTTPHost, "::1")
	t.Setenv(EnvShareHTTPPort, "54321")
	c, err := LoadShareHTTPConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got := c.JoinListenAddr(c.Port); got != "[::1]:54321" {
		t.Fatalf("join: %q", got)
	}
}
