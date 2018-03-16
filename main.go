package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"math/rand"
	"mexs/bots"
	"mexs/common"
	"mexs/exchange"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ConfigFile struct {
	EID       string                     `json:"EID"`
	GA        exchange.AuctionParameters `json:"GA"`
	Ts        int                        `json:"Ts"`
	Days      int                        `json:"Days"`
	SellerIDs []int                      `json:"SellerIDs"`
	BuyerIDs  []int                      `json:"BuyerIDs"`
	// AlgoS and AlgoB is the trading algo used by sellers aad buyers respectively
	AlgoS        []string          `json:"AlgoS"`
	AlgoB        []string          `json:"AlgoB"`
	Sps          []float64         `json:"Sps"`
	Bps          []float64         `json:"Bps"`
	ScheduleType string            `json:"ScheduleType"`
	Info         common.MarketInfo `json:"MarketInfo"`
	Gens         int               `json:"Gens,omitempty"`
	Individuals  int               `json:"Individuals,omitempty"`
	FitnessFN    string            `json:"FitnessFn, omitempty"`
	CInit        string            `json:"CInit, omitempty"`
	EQ           float64           `json:"EQ, omitempty"`
	EP           float64           `json:"EP, omitempty"`
}

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "eid",
			Usage: "Experiment id",
			Value: uuid.New().String(),
		},
		cli.IntFlag{
			Name:  "days",
			Usage: "Number of trading days",
			Value: 3,
		},
		cli.IntFlag{
			Name:  "ts",
			Usage: "Timesteps per trading day",
			Value: 100,
		},
		cli.StringFlag{
			Name:  "config-file",
			Usage: "Configuration file for more complicated experiment setup",
			Value: "NIL",
		},
		cli.IntFlag{
			Name:  "num-sellers",
			Usage: "Set number of sellers",
			Value: 30,
		},
		cli.IntFlag{
			Name:  "num-buyers",
			Usage: "Set number of buyers",
			Value: 30,
		},
		cli.StringFlag{
			Name:  "buyer-algo",
			Usage: "Set buyers algo currently supported [ZIC, ZIP]",
			Value: "ZIP",
		},
		cli.StringFlag{
			Name:  "seller-algo",
			Usage: "Set seller algo currently supported [ZIC, ZIP]",
			Value: "ZIP",
		},
		cli.IntFlag{
			Name:  "blp",
			Usage: "Buyers smallest limit price",
			Value: 5,
		},
		cli.IntFlag{
			Name:  "blps",
			Usage: "Buyers limit price step",
			Value: 1,
		},
		cli.IntFlag{
			Name:  "slp",
			Usage: "Sellers smallest limit price",
			Value: 5,
		},
		cli.IntFlag{
			Name:  "slps",
			Usage: "Sellers limit price step",
			Value: 1,
		},
		cli.StringFlag{
			Name:  "log-level",
			Usage: "Set log level [Debug, Info, Warn, Error]",
			Value: "Info",
		},
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:   "experiment",
			Usage:  "Start a single experiment",
			Action: experiment,
			Flags:  app.Flags,
		},
		cli.Command{
			Name:   "GA",
			Usage:  "Start a evolution process",
			Action: startGA,
			Flags:  app.Flags,
		},
		cli.Command{
			Name:   "ItRun",
			Usage:  "Runs the same market multiple times",
			Action: itRun,
			Flags:  app.Flags,
		},
		cli.Command{
			Name:   "ItGA",
			Usage:  "Runs GA experiment 100 times",
			Action: itGA,
			Flags:  app.Flags,
		},
	}

	app.Name = "Minimal Exchange Simulator"
	app.Usage = "mexs [COMMAND] [OPTIONS]"
	app.Run(os.Args)
}

type ExperimentConfig struct {
	GA          exchange.AuctionParameters
	EID         string
	Ts          int
	Days        int
	SellersIDs  []int
	BuyersIDs   []int
	Agents      map[int]bots.RobotTrader
	Schedule    exchange.AllocationSchedule
	MarketInfo  common.MarketInfo
	Sps         []float64
	Bps         []float64
	Gens        int    `json:"Gens,omitempty"`
	Individuals int    `json:"Individuals,omitempty"`
	FitnessFN   string `json:"FitnessFN, omitempty"`
	CInit       string `json:"CInit, omitempty"`
	EP          float64
	EQ          float64
}

func checkFlags(c *cli.Context) ExperimentConfig {
	configFile := strings.TrimSpace(c.String("config-file"))
	if configFile != "NIL" {
		return getConfigFile(configFile, c)
	}

	log.SetLevel(log.InfoLevel)
	logLevel := strings.TrimSpace(c.String("log-level"))
	switch logLevel {
	case "Debug":
		log.SetLevel(log.DebugLevel)
	case "Info":
		log.SetLevel(log.InfoLevel)
	case "Warn":
		log.SetLevel(log.WarnLevel)
	case "Error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	minSeller := float64(c.Int("slp"))
	step := float64(c.Int("slps"))
	nSellers := c.Int("num-sellers")
	sps := generateSteppedPrices(minSeller, step, 0, nSellers)
	minBuyer := float64(c.Int("blp"))
	step = float64(c.Int("blps"))
	nBuyers := c.Int("num-buyers")
	bps := generateSteppedPrices(minBuyer, step, 0, nBuyers)

	marketInfo := common.MarketInfo{
		MaxPrice:     100.0,
		MinPrice:     1.0,
		MinIncrement: 1,
		MarketEnd:    c.Int("ts"),
		TradingDays:  c.Int("days"),
	}

	traders := make(map[int]bots.RobotTrader)
	traders, sellersIDS := MakeTraders(sps, 0, nSellers, "SELLER", c.String("seller-algo"),
		traders, marketInfo)
	traders, buyersIDS := MakeTraders(bps, nSellers, nBuyers, "BUYER", c.String("buyer-algo"),
		traders, marketInfo)

	GAp := exchange.AuctionParameters{
		BidAskRatio:  1,
		KPricing:     0.5,
		MinIncrement: 1,
		MaxShift:     2,
		WindowSizeEE: 3,
		DeltaEE:      10.0,
		Dominance:    0,
	}

	eConfig := ExperimentConfig{
		GA:         GAp,
		EID:        strings.TrimSpace(c.String("eid")),
		Days:       c.Int("days"),
		Ts:         c.Int("ts"),
		SellersIDs: sellersIDS,
		BuyersIDs:  buyersIDS,
		Agents:     traders,
		MarketInfo: marketInfo,
		Schedule:   generateBasicAllocationSchedule(traders),
		Sps:        sps,
		Bps:        bps,
	}

	return eConfig
}

func MakeTraders(limitPrices []float64, idStart, n int, traderType, traderAlgo string,
	traders map[int]bots.RobotTrader, info common.MarketInfo) (map[int]bots.RobotTrader, []int) {

	ids := make([]int, n)
	orderType := "ASK"
	tType := "SELLER"
	if traderType == "BUYER" {
		orderType = "BID"
		tType = "SELLER"
	}

	if traderAlgo == "ZIP" {
		for i := 0; i < n; i++ {
			zip := &bots.ZIPTrader{}
			zip.InitRobotCore(i+idStart, tType, info)
			zip.AddOrder(&bots.TraderOrder{
				LimitPrice: limitPrices[i],
				Quantity:   1,
				Type:       orderType,
			})
			traders[zip.Info.TraderID] = zip
			ids[i] = zip.Info.TraderID
		}
	} else if traderAlgo == "ZIC" {
		for i := 0; i < n; i++ {
			zip := &bots.ZICTrader{}
			zip.InitRobotCore(i+idStart, tType, info)
			zip.AddOrder(&bots.TraderOrder{
				LimitPrice: limitPrices[i],
				Quantity:   1,
				Type:       orderType,
			})
			traders[zip.Info.TraderID] = zip
			ids[i] = zip.Info.TraderID
		}
	} else {
		log.Panic("Invalid algo type:", traderAlgo)
	}

	return traders, ids
}

func getConfigFile(fileName string, c *cli.Context) ExperimentConfig {
	jsonFile, err := os.Open(fileName)
	if err != nil {
		log.Panic(err.Error())
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var configFile ConfigFile
	json.Unmarshal(byteValue, &configFile)

	log.SetLevel(log.InfoLevel)
	logLevel := strings.TrimSpace(c.String("log-level"))
	switch logLevel {
	case "Debug":
		log.SetLevel(log.DebugLevel)
	case "Info":
		log.SetLevel(log.InfoLevel)
	case "Warn":
		log.SetLevel(log.WarnLevel)
	case "Error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	if configFile.EID == "" {
		configFile.EID = strings.TrimSpace(c.String("eid"))
	}

	traders := make(map[int]bots.RobotTrader)
	for i, id := range configFile.SellerIDs {
		switch configFile.AlgoS[i] {
		case "ZIP":
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "SELLER", configFile.Info)
			zipT.AddOrder(&bots.TraderOrder{
				LimitPrice: configFile.Sps[i],
				Quantity:   1,
				Type:       "ASK",
			})
			traders[zipT.Info.TraderID] = zipT
		case "ZIC":
			zic := &bots.ZICTrader{}
			zic.InitRobotCore(id, "SELLER", configFile.Info)
			zic.AddOrder(&bots.TraderOrder{
				LimitPrice: configFile.Sps[i],
				Quantity:   1,
				Type:       "ASK",
			})
			traders[zic.Info.TraderID] = zic
		default:
			// FIXME: If incorrect type default to ZIP for now
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "SELLER", configFile.Info)
			zipT.AddOrder(&bots.TraderOrder{
				LimitPrice: configFile.Sps[i],
				Quantity:   1,
				Type:       "ASK",
			})
			traders[zipT.Info.TraderID] = zipT
		}
	}

	for i, id := range configFile.BuyerIDs {
		switch configFile.AlgoB[i] {
		case "ZIP":
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "BUYER", configFile.Info)
			zipT.AddOrder(&bots.TraderOrder{
				LimitPrice: configFile.Bps[i],
				Quantity:   1,
				Type:       "BID",
			})
			traders[zipT.Info.TraderID] = zipT
		case "ZIC":
			zic := &bots.ZICTrader{}
			zic.InitRobotCore(id, "BUYER", configFile.Info)
			zic.AddOrder(&bots.TraderOrder{
				LimitPrice: configFile.Bps[i],
				Quantity:   1,
				Type:       "BID",
			})
			traders[zic.Info.TraderID] = zic
		default:
			// FIXME: If incorrect type default to ZIP for now
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "BUYER", configFile.Info)
			zipT.AddOrder(&bots.TraderOrder{
				LimitPrice: configFile.Bps[i],
				Quantity:   1,
				Type:       "BID",
			})
			traders[zipT.Info.TraderID] = zipT
		}
	}

	return ExperimentConfig{
		EID:        configFile.EID,
		GA:         configFile.GA,
		Ts:         configFile.Ts,
		Days:       configFile.Days,
		SellersIDs: configFile.SellerIDs,
		BuyersIDs:  configFile.BuyerIDs,
		MarketInfo: configFile.Info,
		Agents:     traders,
		Sps:        configFile.Sps,
		Bps:        configFile.Bps,
		// For now only standard schedule accepted
		Schedule:    generateBasicAllocationSchedule(traders),
		Gens:        configFile.Gens,
		Individuals: configFile.Individuals,
		FitnessFN:   configFile.FitnessFN,
		CInit:       configFile.CInit,
		EQ:          configFile.EQ,
		EP:          configFile.EP,
	}
}

func experiment(c *cli.Context) {
	eConfig := checkFlags(c)
	log.Debug("Number of traders is:", len(eConfig.Agents))
	ex := exchange.Exchange{}
	ex.Init(eConfig.GA, eConfig.MarketInfo, eConfig.SellersIDs, eConfig.BuyersIDs)
	ex.SetTraders(eConfig.Agents)
	ex.StartMarket(eConfig.EID, eConfig.Schedule)
	supplyAndDemandToCSV(eConfig.Sps, eConfig.Bps, eConfig.EID, "0")
}

// TODO: create a cli interfaces to setup experiment with out having to go
// through the code
// TODO: create tools to automatically create limit prices, orders and
// instantiate traders
// TODO: create a tool for schedule generation

func main1() {
	// NOTE: proof of concept market experiment
	//nOfAgents := 30
	log.Debug("Starting Main")
	rand.Seed(time.Now().UnixNano())
	GAp := exchange.AuctionParameters{
		BidAskRatio:  0.5,
		KPricing:     0.5,
		MinIncrement: 1,
		MaxShift:     1,
		Dominance:    0,
	}

	marketInfo := common.MarketInfo{
		MaxPrice:     100.0,
		MinPrice:     1.0,
		MinIncrement: GAp.MinIncrement,
		MarketEnd:    10,
		TradingDays:  1,
	}

	sellersN := 30
	buyersN := 30
	// NOTE: Use the same for now
	sellerPrices := generateSteppedPrices(5.0, 1, 0, sellersN)
	buyerPrices := generateSteppedPrices(5.0, 1.0, 0, buyersN)
	sellerIDs := make([]int, sellersN)
	buyerIDs := make([]int, buyersN)

	log.Debug("SellerPrices:", sellerPrices)
	log.Debug("BuyerPrices:", buyerPrices)

	traders := make(map[int]bots.RobotTrader)
	for i := 0; i < buyersN; i++ {
		zip := &bots.ZIPTrader{}
		zip.InitRobotCore(i, "BUYER", marketInfo)
		zip.AddOrder(&bots.TraderOrder{
			LimitPrice: buyerPrices[i],
			Quantity:   1,
			Type:       "BID",
		})
		traders[zip.Info.TraderID] = zip
		buyerIDs[i] = zip.Info.TraderID
	}

	for i := 0; i < sellersN; i++ {
		zip := &bots.ZIPTrader{}
		zip.InitRobotCore(i+buyersN, "SELLER", marketInfo)
		zip.AddOrder(&bots.TraderOrder{
			LimitPrice: sellerPrices[i],
			Quantity:   1,
			Type:       "ASK",
		})
		traders[zip.Info.TraderID] = zip
		sellerIDs[i] = zip.Info.TraderID
	}

	s := generateBasicAllocationSchedule(traders)

	ex := exchange.Exchange{}
	ex.Init(GAp, marketInfo, sellerIDs, buyerIDs)
	ex.SetTraders(traders)
	experimentID := uuid.New()
	ex.StartMarket(experimentID.String(), s)
	supplyAndDemandToCSV(sellerPrices, buyerPrices, experimentID.String(), "1")
}

// generateSteppedPrices creates limit prices
// @param min :- minimum value
// @param noise :- rand [-noise, ..., noise] added to values
// @param n is the number of prices to generate
func generateSteppedPrices(min, step float64, noise, n int) []float64 {
	prices := make([]float64, n)
	if noise != 0 {

		for i := 0; i < n; i++ {
			prices[i] = min + float64(i)*step + float64(rand.Intn(2*noise)-noise)
		}

		return prices
	}

	for i := 0; i < n; i++ {
		prices[i] = min + float64(i)*step
	}

	return prices
}

func generateBasicAllocationSchedule(traders map[int]bots.RobotTrader) exchange.AllocationSchedule {
	// This generates an allocation schedule in which all traders get the same order each training day

	s := exchange.AllocationSchedule{
		Schedule: make(map[int]map[int][]exchange.TID2RTO),
	}
	s.Schedule[-1] = make(map[int][]exchange.TID2RTO)
	s.Schedule[-1][0] = make([]exchange.TID2RTO, 0)

	for k, v := range traders {
		tido := exchange.TID2RTO{
			TraderID:  k,
			ExecOrder: v.GetExecutionOrder(),
		}
		s.Schedule[-1][0] = append(s.Schedule[-1][0], tido)
	}
	return s
}

func supplyAndDemandToCSV(sellers, buyers []float64, experimentID string, number string) {
	sort.Sort(float64arr(sellers))
	sort.Sort(sort.Reverse(float64arr(buyers)))

	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/LIMITPRICES_%s.csv", experimentID, number))
	if err != nil {
		log.WithFields(log.Fields{
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("File Path not found")
		return
	}

	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"experimentID": experimentID,
			"error":        err.Error(),
		}).Error("Limit prices CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Type 1 is ask, type 2 is bid
	writer.Write([]string{"NUMBER", "TYPE", "LIMIT_PRICE"})
	for _, v := range sellers {
		writer.Write([]string{
			number,
			"ASK",
			fmt.Sprintf("%.3f", v),
		})
	}

	for _, v := range buyers {
		writer.Write([]string{
			number,
			"BID",
			fmt.Sprintf("%.3f", v),
		})
	}

	log.Debug("Trades saved to file:", fileName)
}

type float64arr []float64

func (a float64arr) Len() int           { return len(a) }
func (a float64arr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a float64arr) Less(i, j int) bool { return a[i] < a[j] }

func startGA(c *cli.Context) {
	config := checkFlags(c)

	ga := &GA{
		N:                   config.Individuals,
		Gens:                config.Gens,
		Config:              config,
		CurrentGen:          0,
		EquilibriumQuantity: config.EQ,
		EquilibriumPrice:    config.EP,
	}

	ga.Start()
}

func itRun(c *cli.Context) {
	runs := 100
	for i := 0; i < runs; i++ {
		log.Warn("Run:", i)
		config := checkFlags(c)
		ex := exchange.Exchange{}
		ex.Init(config.GA, config.MarketInfo, config.SellersIDs, config.BuyersIDs)
		ex.SetTraders(config.Agents)
		ex.StartMarket(config.EID+"_"+strconv.Itoa(i), config.Schedule)
		supplyAndDemandToCSV(config.Sps, config.Bps, config.EID+"_"+strconv.Itoa(i), "0")
	}
}

func itGA(c *cli.Context) {
	runs := 100
	for i:= 0; i < runs; i++ {
		log.Warn("Run: ", i)
		config := checkFlags(c)
		config.EID = config.EID +"/run_"+strconv.Itoa(i)
		ga := &GA{
			N:                   config.Individuals,
			Gens:                config.Gens,
			Config:              config,
			CurrentGen:          0,
			EquilibriumQuantity: config.EQ,
			EquilibriumPrice:    config.EP,
		}
		ga.Start()
	}
}