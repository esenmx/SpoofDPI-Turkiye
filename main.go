package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/esenmx/SpoofDPI-Turkiye/proxy"
	"github.com/esenmx/SpoofDPI-Turkiye/util"
	"github.com/esenmx/SpoofDPI-Turkiye/util/log"
	"github.com/esenmx/SpoofDPI-Turkiye/version"
)

func main() {
	os.Exit(run())
}

func run() int {
	args := util.ParseArgs()
	if args.Version {
		version.PrintVersion()
		return 0
	}

	config := util.GetConfig()
	if err := config.Load(args); err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		return 2
	}

	log.InitLogger(config.Debug)
	ctx, cancel := context.WithCancel(log.GetCtxWithScope(context.Background(), "MAIN"))
	defer cancel()
	logger := log.GetCtxLogger(ctx)

	pxy := proxy.New(config)

	if !config.Silent {
		util.PrintColoredBanner()
	}

	if config.SystemProxy {
		if err := util.SetOsProxy(uint16(config.Port)); err != nil {
			logger.Error().Msgf("error while changing proxy settings: %s", err)
			return 1
		}
		defer func() {
			if err := util.UnsetOsProxy(); err != nil {
				logger.Error().Msgf("error while disabling proxy: %s", err)
			}
		}()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pxy.Start(ctx)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	select {
	case sig := <-sigs:
		logger.Info().Msgf("received %s, shutting down", sig)
		cancel()
		<-errCh
		return 0
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error().Msgf("proxy exited: %s", err)
			return 1
		}
		return 0
	}
}
