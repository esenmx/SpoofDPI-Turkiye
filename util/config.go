package util

import (
	"fmt"
	"regexp"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

type Config struct {
	Addr            string
	Port            int
	DnsAddr         string
	DnsPort         int
	DnsFallback     []string
	DnsIPv4Only     bool
	EnableDoh       bool
	DohUrl          string
	DohBootstrapIp  string
	Debug           bool
	Silent          bool
	SystemProxy     bool
	Timeout         int
	WindowSize      int
	AllowedPatterns []*regexp.Regexp
}

var config *Config

func GetConfig() *Config {
	if config == nil {
		config = new(Config)
	}
	return config
}

func (c *Config) Load(args *Args) error {
	c.Addr = args.Addr
	c.Port = int(args.Port)
	c.DnsAddr = args.DnsAddr
	c.DnsPort = int(args.DnsPort)
	c.DnsFallback = []string(args.DnsFallback)
	c.DnsIPv4Only = args.DnsIPv4Only
	c.Debug = args.Debug
	c.EnableDoh = args.EnableDoh
	c.DohUrl = args.DohUrl
	c.DohBootstrapIp = args.DohBootstrapIp
	c.Silent = args.Silent
	c.SystemProxy = args.SystemProxy
	c.Timeout = int(args.Timeout)
	c.WindowSize = int(args.WindowSize)

	patterns, err := compileAllowedPatterns(args.AllowedPattern)
	if err != nil {
		return err
	}
	c.AllowedPatterns = patterns
	return nil
}

func compileAllowedPatterns(patterns StringArray) ([]*regexp.Regexp, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid -pattern %q: %w", pattern, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func PrintColoredBanner() {
	cyan := putils.LettersFromStringWithStyle("Spoof", pterm.NewStyle(pterm.FgCyan))
	purple := putils.LettersFromStringWithStyle("DPI", pterm.NewStyle(pterm.FgLightMagenta))
	pterm.DefaultBigText.WithLetters(cyan, purple).Render()

	pterm.DefaultBulletList.WithItems([]pterm.BulletListItem{
		{Level: 0, Text: "ADDR    : " + fmt.Sprint(config.Addr)},
		{Level: 0, Text: "PORT    : " + fmt.Sprint(config.Port)},
		{Level: 0, Text: "DNS     : " + fmt.Sprint(config.DnsAddr)},
		{Level: 0, Text: "DEBUG   : " + fmt.Sprint(config.Debug)},
	}).Render()

	pterm.DefaultBasicText.Println("Spoof DPI'ın bu sürümü Türkiye'de kullanılmak üzere yapılandırılmıştır.")
	pterm.DefaultBasicText.Println("Çıkmak için 'CTRL + c' tuş kombinasyonunu kullanın.")
}
