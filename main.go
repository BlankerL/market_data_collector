package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/beaquant/utils/json_file"
	"github.com/goex-top/market_center"
	mcc "github.com/goex-top/market_center_client"
	"github.com/goex-top/market_data_collector/client"
	"github.com/goex-top/market_data_collector/collector"
	"github.com/goex-top/market_data_collector/config"
	"github.com/goex-top/market_data_collector/storage"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	cfg config.Config
)

func usage() {
	fmt.Fprintf(os.Stderr, `market data collector version: v1.0.0
Usage: market_data_collector [-h] [-c config.json]
Options:
`)
	flag.PrintDefaults()
}

func main() {
	var c string
	var help bool
	flag.StringVar(&c, "c", "config.json", "set configuration `json file`")
	flag.BoolVar(&help, "h", false, "this help")
	flag.Usage = usage

	flag.Parse()
	if help {
		flag.Usage()
		return
	}

	err := json_file.Load(c, &cfg)
	if err != nil {
		panic(err)
	}

	if !cfg.Store.Csv {
		panic("currently only support csv, please check your configure")
	}
	log.Println(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	for _, v := range cfg.Subs {
		sto := storage.NewCsvStorage(ctx, v.ExchangeName, v.CurrencyPair, v.Flag, "csv", "tar")
		go sto.SaveWorker()
		cl := &client.Client{}
		if v.Direct {
			cl = client.NewClient(v.ExchangeName, v.CurrencyPair, "", nil)
		} else {
			mccc := mcc.NewClient()
			isSpot := market_center.IsFutureExchange(v.ExchangeName)
			if v.Flag&market_center.DataFlag_Depth != 0 {
				if isSpot {
					mccc.SubscribeSpotDepth(v.ExchangeName, v.CurrencyPair, v.Period)
				} else {
					mccc.SubscribeFutureDepth(v.ExchangeName, v.ContractType, v.CurrencyPair, v.Period)
				}
			}
			if v.Flag&market_center.DataFlag_Ticker != 0 {
				if isSpot {
					mccc.SubscribeSpotTicker(v.ExchangeName, v.CurrencyPair, v.Period)
				} else {
					mccc.SubscribeFutureTicker(v.ExchangeName, v.ContractType, v.CurrencyPair, v.Period)
				}
			}
			cl = client.NewClient(v.ExchangeName, v.CurrencyPair, "", mccc)
		}

		collector.NewCollector(ctx, cl, v.Period, v.Flag, sto)
	}

	exitSignal := make(chan os.Signal, 1)
	sigs := []os.Signal{os.Interrupt, syscall.SIGILL, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGTERM}
	signal.Notify(exitSignal, sigs...)
	<-exitSignal
	cancel()
	time.Sleep(time.Second)
	log.Println("market data collector exit")
}
