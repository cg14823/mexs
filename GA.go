package main

import (
	"encoding/csv"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"mexs/exchange"
	"os"
	"path/filepath"
	"strconv"
)

type tradesCSV struct {
	ID  int
	TD  int
	TS  int
	P   float64
	SID int
	BID int
	AP  float64
	BP  float64
}

type GA struct {
	// Number of individuals in each gen
	N            int
	Gens         int
	Config       ExperimentConfig
	CurrentGen   int
	currentGenes []exchange.AuctionParameters
	// Some way of mapping equilibrium price over time is needed
	EquilibriumPrice    float64
	EquilibriumQuantity float64
}

func (g *GA) Start() {
	// This function will be the heart of the GA
	// Number of individuals in each generation
	err := os.MkdirAll("../mexs/logs/"+g.Config.EID+"/", 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Log Folder for this experiment could not be made")
	}
	supplyAndDemandToCSV(g.Config.Sps, g.Config.Bps, g.Config.EID, "0")
	// Things that remain constant per generation

	// START BY initializing the chromosomes
	cs := make([]exchange.AuctionParameters, g.N)
	for i := 0; i < g.N; i++ {

		cs[i] = InitializeChromozones("LOW")
	}
	g.currentGenes = cs

	for i := 0; i < g.Gens; i++ {
		log.Warn("GEN:", i)

		err := os.MkdirAll("../mexs/logs/"+g.Config.EID+"/GEN_"+strconv.Itoa(i)+"/", 0755)
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Error("Log Folder for this generation could not be made")
		}

		g.MakeGen(g.currentGenes, strconv.Itoa(i))
		scores := g.FitnessFunction("ALPHA", strconv.Itoa(i))
		g.chromozonesToCSV(i, g.currentGenes, scores)
		g.createNewGen(scores)
	}

}
func (g *GA) createNewGen(scores []float64) {
	for i := 0; i < g.N; i++ {
		g.currentGenes[i] = g.getChildGenes(scores)
	}
}

func (g *GA) getChildGenes(scores []float64) exchange.AuctionParameters {
	contenders := []int{rand.Intn(g.N), rand.Intn(g.N), rand.Intn(g.N), rand.Intn(g.N), rand.Intn(g.N)}
	// ix1 -> MOM
	ix1 := 0
	// ix2 -> DAD
	ix2 := 0
	maxSc := scores[contenders[ix1]]
	for i := 1; i < len(contenders); i++ {
		if scores[contenders[i]] > maxSc {
			maxSc = scores[contenders[i]]
			ix2 = ix1
			ix1 = i
		}
	}

	var kp float64
	var minI float64
	var maxS float64
	var dom int

	// NOTE: P(DAD) = 0.5 P(MOM) = 0.5
	// KPricing mutation is in range of [-0.05, 0.05] with limits [0,1]
	if val := rand.Intn(10); val < 5 {
		// MOM
		kp = g.currentGenes[ix1].KPricing + float64(rand.Intn(6+6)-6)/100.0
	} else {
		//DAD
		kp = g.currentGenes[ix2].KPricing + float64(rand.Intn(6+6)-6)/100.0
	}

	if kp < 0 {
		kp = 0.0
	}

	if kp > 1 {
		kp = 1.0
	}
	// MinIncrement mutation is in range of [-0.5, 05] with limit [0, 20]
	if val := rand.Intn(10); val < 5 {
		// MOM
		minI = g.currentGenes[ix1].MinIncrement + float64(rand.Intn(6000+6000)-6000)/10000.0
	} else {
		//DAD
		minI = g.currentGenes[ix2].MinIncrement + float64(rand.Intn(6000+6000)-600)/100.0
	}
	if minI < 0 {
		minI = 0.0
	}
	if minI > 20 {
		minI = 20.0
	}
	// MaxShift mutation is in range of [-0.01, 0.01] with limit [0.05, 10]
	if val := rand.Intn(10); val < 5 {
		// MOM
		maxS = g.currentGenes[ix1].MaxShift + float64(rand.Intn(11000+11000)-11000)/1000000.0
	} else {
		//DAD
		maxS = g.currentGenes[ix2].MaxShift + float64(rand.Intn(11000+6000)-11000)/1000000.0
	}
	if maxS < 0.05 {
		maxS = 0.05
	}
	if maxS > 10 {
		maxS = 10.0
	}

	// Dominance mutation is in range of [-1, 1] with limit [0, 10]
	if val := rand.Intn(10); val < 5 {
		// MOM
		dom = g.currentGenes[ix1].Dominance + rand.Intn(2+1) - 1
		//DAD
		dom = g.currentGenes[ix2].Dominance + rand.Intn(2+1) - 1
	}
	if dom < 0 {
		dom = 0
	}
	if dom > 10 {
		dom = 10
	}
	bar := g.currentGenes[ix1].BidAskRatio + float64(rand.Intn(200+200)-200)/10000.0
	if bar > 10.0 {
		bar = 10.0
	}
	if bar < 0.1 {
		bar = 0.1
	}
	return exchange.AuctionParameters{
		BidAskRatio:  bar,
		KPricing:     kp,
		MinIncrement: minI,
		MaxShift:     maxS,
		Dominance:    dom,
		OrderQueuing: 1,
	}
}

func (g *GA) scoresToCSV(scores []float64, gen int) {
	// TODO:
}

func (g *GA) MakeGen(cs []exchange.AuctionParameters, gen string) {
	for i := 0; i < g.N; i++ {
		ex := &exchange.Exchange{}
		ex.Init(cs[i], g.Config.MarketInfo, g.Config.SellersIDs, g.Config.BuyersIDs)
		ex.SetTraders(g.Config.Agents)
		ex.StartMarket(g.Config.EID+"/GEN_"+gen+"/IND_"+strconv.Itoa(i), g.Config.Schedule)
	}
}

func (g *GA) FitnessFunction(fnName string, gen string) []float64 {
	// Allow for different functions to be used
	switch fnName {
	case "ALPHA":
		// 1 / alpha is the score so that higher is better
		trades := g.readTradesCSV(fmt.Sprintf("../mexs/logs/%s/GEN_%s/", g.Config.EID, gen))
		return g.allAlphaScores(trades)

	case "ALOC-EFF":
		return []float64{}
	case "AVG-TRADER-EFF":
		return []float64{}
	case "COM-EFFICENCY":
		return []float64{}
	default:
		// default to whatever you prefer
		return []float64{}
	}
}

func (g *GA) allAlphaScores(trades map[int][]tradesCSV) []float64 {
	scores := make([]float64, g.N)
	for k, v := range trades {
		scores[k] = g.alphaFitnessFn(v)
	}
	return scores
}

func (g *GA) alphaFitnessFn(trades []tradesCSV) float64 {
	tNum := float64(len(trades))
	if tNum == 0 {
		return 0.0
	}

	alpha := 100.0 / g.EquilibriumPrice
	sum := 0.0

	for _, t := range trades {
		sum += math.Pow(t.P-g.EquilibriumPrice, 2.0)
	}
	sum = sum / tNum
	alpha *= math.Sqrt(sum)
	// Invert so that the higher the number the better the score
	return 1.0 / alpha
}

func (g *GA) readTradesCSV(folderPath string) map[int][]tradesCSV {
	allTrades := make(map[int][]tradesCSV)
	for i := 0; i < g.N; i++ {
		fileName, err := filepath.Abs(folderPath + "IND_" + strconv.Itoa(i) + "/TRADES.csv")
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Panic("Could not find path to trade file: ", fileName)
		}

		file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
		lines, err := csv.NewReader(file).ReadAll()
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Panic("Could not read the file:", fileName)
		}
		file.Close()

		// remove headers from csv
		lines = lines[1:][:]
		trades := make([]tradesCSV, len(lines))
		for i, line := range lines {
			id, _ := strconv.Atoi(line[0])
			td, _ := strconv.Atoi(line[1])
			ts, _ := strconv.Atoi(line[2])
			p, _ := strconv.ParseFloat(line[3], 64)
			sid, _ := strconv.Atoi(line[4])
			bid, _ := strconv.Atoi(line[5])
			ap, _ := strconv.ParseFloat(line[6], 64)
			bp, _ := strconv.ParseFloat(line[7], 64)

			trades[i] = tradesCSV{
				ID:  id,
				TD:  td,
				TS:  ts,
				P:   p,
				SID: sid,
				BID: bid,
				AP:  ap,
				BP:  bp,
			}
		}
		allTrades[i] = trades
	}
	return allTrades
}

func (g *GA) chromozonesToCSV(gen int, cs []exchange.AuctionParameters, scores []float64) {
	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/chromozones.csv", g.Config.EID))
	if err != nil {
		log.WithFields(log.Fields{
			"experimentID": g.Config.EID,
			"error":        err.Error(),
		}).Error("File Path not found")
		return
	}
	addHeader := true
	if _, err := os.Stat(fileName); err == nil {
		addHeader = false
	}

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if addHeader {
		writer.Write([]string{
			"Gen",
			"Score",
			"B:A",
			"K",
			"MinIncrement",
			"MaxShift",
			"Dominance",
		})
	}

	for i, v := range cs {
		writer.Write([]string{
			strconv.Itoa(gen),
			fmt.Sprintf("%.5f", scores[i]),
			fmt.Sprintf("%.5f", v.BidAskRatio),
			fmt.Sprintf("%.5f", v.KPricing),
			fmt.Sprintf("%.5f", v.MinIncrement),
			fmt.Sprintf("%.5f", v.MaxShift),
			strconv.Itoa(v.Dominance),
		})
	}
}

func InitializeChromozones(initType string) exchange.AuctionParameters {
	switch initType {
	case "LOW":
		return exchange.AuctionParameters{
			BidAskRatio:  0.1,
			KPricing:     0,
			MinIncrement: 0,
			MaxShift:     0.1,
			Dominance:    0,
			OrderQueuing: 1,
		}
	case "NORMAL":
		return exchange.AuctionParameters{
			BidAskRatio:  1,
			KPricing:     0.5,
			MinIncrement: 1,
			MaxShift:     2,
			Dominance:    0,
			OrderQueuing: 1,
		}
	case "HIGH":
		return exchange.AuctionParameters{
			BidAskRatio:  10,
			KPricing:     1,
			MinIncrement: 10,
			MaxShift:     100,
			Dominance:    10,
			OrderQueuing: 1,
		}
	case "RANDOM":
		// random, in logical range
		// BidAsk Ratio between [0.5, 1.5)
		// KPricing between [0,1)
		// MinIncrement between [0,5)
		// MaxShift between [0.5,1.5)
		// Domminance between [0,5)
		return exchange.AuctionParameters{
			BidAskRatio:  float64(rand.Intn(200+50)+50) / 100.0,
			KPricing:     rand.Float64(),
			MinIncrement: float64(rand.Intn(5)),
			MaxShift:     rand.Float64() + 0.5,
			Dominance:    rand.Intn(5),
			OrderQueuing: 1,
		}
	default:
		// Note: default to any especial case I want to test out
		return exchange.AuctionParameters{
			BidAskRatio:  float64(rand.Intn(200+50)+50) / 100.0,
			KPricing:     rand.Float64(),
			MinIncrement: float64(rand.Intn(5)),
			MaxShift:     rand.Float64() + 0.5,
			Dominance:    rand.Intn(5),
			OrderQueuing: 1,
		}
	}
}
