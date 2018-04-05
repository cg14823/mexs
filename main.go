package main

import (
	"encoding/json"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"mexs/bots"
	"mexs/common"
	"mexs/exchange"
	"os"
	"strconv"
	"strings"
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
	ScheduleType string            `json:"ScheduleType"`
	Info         common.MarketInfo `json:"MarketInfo"`
	Gens         int               `json:"Gens,omitempty"`
	Individuals  int               `json:"Individuals,omitempty"`
	FitnessFN    string            `json:"FitnessFn, omitempty"`
	CInit        string            `json:"CInit, omitempty"`
	EQ           float64           `json:"EQ, omitempty"`
	EP           float64           `json:"EP, omitempty"`
	SandDs map[int]exchange.SandD
	Sched []exchange.SchedToPrices `json:"Schedule, omitempty"`
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
	SandDs map[int]exchange.SandD
	SLP []exchange.AgentLimitPrices `json:"SLimitPrices, omitempty"`
	BLP []exchange.AgentLimitPrices `json:"BLimitPrices, omitempty"`
}

func checkFlags(c *cli.Context) ExperimentConfig {
	configFile := strings.TrimSpace(c.String("config-file"))
	if configFile != "NIL" {
		return getConfigFile(configFile, c)
	} else{
		log.Panic("A configuration file is required to be passed in use flag --config-file")
		return ExperimentConfig{}
	}

	// FIXME ONLY RUNNABLE THOUGH CONFIG FILE NOW

	//log.SetLevel(log.InfoLevel)
	//logLevel := strings.TrimSpace(c.String("log-level"))
	//switch logLevel {
	//case "Debug":
	//	log.SetLevel(log.DebugLevel)
	//case "Info":
	//	log.SetLevel(log.InfoLevel)
	//case "Warn":
	//	log.SetLevel(log.WarnLevel)
	//case "Error":
	//	log.SetLevel(log.ErrorLevel)
	//default:
	//	log.SetLevel(log.InfoLevel)
	//}
	//
	//minSeller := float64(c.Int("slp"))
	//step := float64(c.Int("slps"))
	//nSellers := c.Int("num-sellers")
	//sps := generateSteppedPrices(minSeller, step, 0, nSellers)
	//minBuyer := float64(c.Int("blp"))
	//step = float64(c.Int("blps"))
	//nBuyers := c.Int("num-buyers")
	//bps := generateSteppedPrices(minBuyer, step, 0, nBuyers)
	//
	//marketInfo := common.MarketInfo{
	//	MaxPrice:     100.0,
	//	MinPrice:     1.0,
	//	MinIncrement: 1,
	//	MarketEnd:    c.Int("ts"),
	//	TradingDays:  c.Int("days"),
	//}
	//
	//traders := make(map[int]bots.RobotTrader)
	//traders, sellersIDS := MakeTraders(sps, 0, nSellers, "SELLER", c.String("seller-algo"),
	//	traders, marketInfo)
	//traders, buyersIDS := MakeTraders(bps, nSellers, nBuyers, "BUYER", c.String("buyer-algo"),
	//	traders, marketInfo)
	//
	//GAp := exchange.AuctionParameters{
	//	BidAskRatio:  1,
	//	KPricing:     0.5,
	//	MinIncrement: 1,
	//	MaxShift:     2,
	//	WindowSizeEE: 3,
	//	DeltaEE:      10.0,
	//	Dominance:    0,
	//}
	//
	//sched, sand := generateBasicAllocationSchedule(sellersIDS, buyersIDS, sps, bps, marketInfo.TradingDays)
	//eConfig := ExperimentConfig{
	//	GA:         GAp,
	//	EID:        strings.TrimSpace(c.String("eid")),
	//	Days:       c.Int("days"),
	//	Ts:         c.Int("ts"),
	//	SellersIDs: sellersIDS,
	//	BuyersIDs:  buyersIDS,
	//	Agents:     traders,
	//	MarketInfo: marketInfo,
	//	Schedule:   sched,
	//	SandDs: sand,
	//	Sps:        sps,
	//	Bps:        bps,
	//}
	//
	//return eConfig
}

func MakeTraders(limitPrices []float64, idStart, n int, traderType, traderAlgo string,
	traders map[int]bots.RobotTrader, info common.MarketInfo) (map[int]bots.RobotTrader, []int) {
	ids := make([]int, n)
	tType := "SELLER"
	if traderType == "BUYER" {
		tType = "SELLER"
	}

	if traderAlgo == "ZIP" {
		for i := 0; i < n; i++ {
			zip := &bots.ZIPTrader{}
			zip.InitRobotCore(i+idStart, tType, info)
			traders[zip.Info.TraderID] = zip
			ids[i] = zip.Info.TraderID
		}
	} else if traderAlgo == "ZIC" {
		for i := 0; i < n; i++ {
			zip := &bots.ZICTrader{}
			zip.InitRobotCore(i+idStart, tType, info)
			traders[zip.Info.TraderID] = zip
			ids[i] = zip.Info.TraderID
		}
	} else if traderAlgo == "AA" {
		for i := 0; i < n; i++ {
			t := &bots.AATrader{}
			t.InitRobotCore(i+idStart, tType, info)
			traders[t.Info.TraderID] = t
			ids[i] = t.Info.TraderID
		}
	}else {
		log.Panic("Invalid algo type:", traderAlgo)
	}

	return traders, ids
}

func getConfigFile(fileName string, c *cli.Context) ExperimentConfig {
	jsonFile, err := os.Open(fileName)
	if err != nil {
		log.Panic(err.Error())
	}

	// Parse json into ConfigFile struct
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var configFile ConfigFile
	json.Unmarshal(byteValue, &configFile)

	// Set log level only set through command line
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

	// Generate experiment id
	if configFile.EID == "" {
		configFile.EID = strings.TrimSpace(c.String("eid"))
	}

	// Create the agents for the experiment
	traders := make(map[int]bots.RobotTrader)
	for i, id := range configFile.SellerIDs {
		switch configFile.AlgoS[i] {
		case "ZIP":
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "SELLER", configFile.Info)
			traders[zipT.Info.TraderID] = zipT
		case "ZIC":
			zic := &bots.ZICTrader{}
			zic.InitRobotCore(id, "SELLER", configFile.Info)
			traders[zic.Info.TraderID] = zic
		case "AA":
			aa := &bots.AATrader{}
			aa.InitRobotCore(id, "SELLER", configFile.Info)
			traders[aa.Info.TraderID] = aa
		default:
			log.Panic("SHIIT")
		}
	}

	for i, id := range configFile.BuyerIDs {
		switch configFile.AlgoB[i] {
		case "ZIP":
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "BUYER", configFile.Info)
			traders[zipT.Info.TraderID] = zipT
		case "ZIC":
			zic := &bots.ZICTrader{}
			zic.InitRobotCore(id, "BUYER", configFile.Info)
			traders[zic.Info.TraderID] = zic
		case "AA":
			aa := &bots.AATrader{}
			aa.InitRobotCore(id, "BUYER", configFile.Info)
			traders[aa.Info.TraderID] = aa
		default:
			log.Panic("SHIIT")
		}
	}

	sched, sand := generateSchedule(configFile.ScheduleType,configFile.SellerIDs, configFile.BuyerIDs, configFile.Sched, configFile.Days)
	return ExperimentConfig{
		EID:        configFile.EID,
		GA:         configFile.GA,
		Ts:         configFile.Ts,
		Days:       configFile.Days,
		SellersIDs: configFile.SellerIDs,
		BuyersIDs:  configFile.BuyerIDs,
		MarketInfo: configFile.Info,
		Agents:     traders,
		// For now only standard schedule accepted
		Schedule:    sched,
		SandDs: sand,
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
	ex.StartMarket(eConfig.EID, eConfig.Schedule, eConfig.SandDs)
}


// Create schedule based on config file
// For now only standard supported
// Standard schedule all traders get the same units at the start of the trading day
// To be supported [MarketShock, stepped]
func generateSchedule(schedType string, sellerIDs, buyerIDs []int, schedAndPrices []exchange.SchedToPrices, days int) (exchange.AllocationSchedule, map[int]exchange.SandD) {
	switch schedType {
	case "STANDARD":
		return generateStandardSched(sellerIDs, buyerIDs, schedAndPrices[0], days)
	default:
		log.WithFields(log.Fields{
			"Valid options": "[STANDARD]",
			"Given option": schedType,
		}).Panic("The schedule type is unsupported")
		return exchange.AllocationSchedule{}, make(map[int]exchange.SandD)
	}
}

// Standard schedule has only one set of limit prices that are refilled each day
func generateStandardSched(sellerIDs, buyerIDs []int, schedAndPrices exchange.SchedToPrices, days int)(exchange.AllocationSchedule, map[int]exchange.SandD) {
	allocSched := exchange.AllocationSchedule{
		Schedule: make(map[int]map[int]int),
	}

	for d:=0; d < days; d++ {
		allocSched.Schedule[d] = make(map[int]int)
		allocSched.Schedule[d][0] = schedAndPrices.SID
	}


	sandd := exchange.SandD{
		ID: schedAndPrices.SID,
		SIDs: sellerIDs,
		BIDs: buyerIDs,
		Sps: schedAndPrices.SLimitPrices,
		Bps: schedAndPrices.BLimitPrices,
	}

	sMap := make(map[int]exchange.SandD)
	sMap[schedAndPrices.SID] = sandd

	return allocSched, sMap
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
		MutationRate: 0.1,
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
		ex.StartMarket(config.EID+"_"+strconv.Itoa(i), config.Schedule, config.SandDs)
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
			MutationRate: 0.06,
		}
		ga.Start()
	}
}