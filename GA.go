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
	"errors"
	"sort"
)

type tradesCSV struct {
	// Trade ID
	ID int
	// Trading day
	TD int
	// Time step
	TS int
	// Price
	P float64
	// Seller ID
	SID int
	// Bid ID
	BID int
	// Ask price
	AP float64
	// Bid price
	BP float64
}

type tradeLPs struct {
	// trader id
	TID int
	// time step
	TS int
	// trading day
	TD int
	// trade price
	TP float64
	// seller limit price
	Slp float64
	// buyer limit price
	Blp float64
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
	log.WithFields(log.Fields{
		"EID": g.Config.EID,
		"Individuals": g.N,
		"Gens": g.Gens,
	}).Warn("STARTING GA")
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
		scores := g.FitnessFunction("ALOC-EFF", strconv.Itoa(i))
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

	mom := g.currentGenes[contenders[ix1]]
	dad := g.currentGenes[contenders[ix2]]

	var kp float64
	var minI float64
	var win int
	var delta float64
	var maxS float64
	var dom int
	// KPricing mutation is in range of [-0.05, 0.05] with limits [0,1]
	kp = mutateFloat(mom.KPricing, dad.KPricing, 0.0, 1.0, 5, -5, 50,3, 5 )

	// MinIncrement mutation is in range of [-0.5, 05] with limit [0, 20]
	minI = mutateFloat(mom.KPricing, dad.KPricing, 0.0, 1.0, 5, -5, 50,3, 4 )

	// WindowSizeEE mutation is in range of [-1, +1] with limit [1, 20]
	win = mutateInt(mom.WindowSizeEE, dad.WindowSizeEE, 2, -1, 50, 1, 20)

	// DeltaEE mutation is in range of [-1.0, +1] with limit [0, 100]
	delta = mutateFloat(mom.DeltaEE, dad.DeltaEE, 0.0, 100.0, 1, -1, 50, 3, 3)

	// MaxShift mutation is in range of [-0.01, 0.01] with limit [0.05, 10]
	maxS = mutateFloat(mom.MaxShift, dad.MaxShift, 0.05, 10, 1,-1,50, 3,5)

	// Dominance mutation is in range of [-1, 1] with limit [0, 10]
	dom = mutateInt(mom.Dominance, dad.Dominance, 2,-1, 50, 0,10)

	// BidAsk ratio mutation always mom gene  range [ -0.05, 0.05] with limit [0.2, 5]
	bar := mutateFloat(mom.BidAskRatio, mom.BidAskRatio, 0.2, 5.0, 5, -5, 0,3, 5)
	return exchange.AuctionParameters{
		BidAskRatio:  bar,
		KPricing:     kp,
		MinIncrement: minI,
		WindowSizeEE: win,
		DeltaEE: delta,
		MaxShift:     maxS,
		Dominance:    dom,
		OrderQueuing: 1,
	}
}
// @param mom - mothers gene
// @param dad - dads gene
// @param max - max mutation value
// @param min - min mutation value
// @param prob - probability of choosing mom ( [0, 100]
// @param ubound - upper bound in gene value
// @param lbound - lower bound in gene value
func mutateInt(mom, dad, max, min, prob, lbound, ubound int) int {
	mutation := rand.Intn(max - min) + min
	v := dad + mutation
	if val := rand.Intn(100); val < prob {
		v = mom + mutation
	}

	if v < lbound {
		return lbound
	} else if  v > ubound{
		return ubound
	}
	return v
}

// @param mom - mothers gene
// @param dad - dads gene
// @param max - max mutation value
// @param min - min mutation value
// @param prob - probability of choosing mom ( [0, 100]
// To create random mutations up to x decimals we multiply mav values and min values by
// 10 * decimals and then use (rand.Int((max - min) * 10 ** d1) -min *10**d1) / (10 *d2)
func mutateFloat(mom, dad, lbound, ubound  float64, max, min, prob, d1, d2 int) float64 {
	nMax := max * int(math.Pow(10.0, float64(d1)))
	nMin := min * int(math.Pow(10.0, float64(d1)))
	randN := float64(rand.Intn(nMax - nMin) + nMin)
	mutation :=  randN / math.Pow(10.0, float64(d2))
	v := dad +  mutation
	if val := rand.Intn(100); val < prob {
		v = mom + mutation
	}

	if v < lbound {
		return lbound
	} else if  v > ubound{
		return ubound
	}
	return v
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
		// FIXME: assume no market shocks for now
		// Regular schedule
		pe, _, err := calculateEQ(g.Config.Sps, g.Config.Bps)
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
				"pe": pe,
			}).Panic("Failed to calculates EQ in fitness function")
		}
		sS, bS := calculateMaxSurplus(g.Config.Sps, g.Config.Bps, pe)
		surplusDays := (sS +bS) * float64(g.Config.MarketInfo.TradingDays)
		log.WithFields(log.Fields{
			"surplusDays": surplusDays,
			"Pe": pe,
		}).Info("Efficency data")
		trades := g.getLimitPrices(fmt.Sprintf("../mexs/logs/%s/GEN_%s/", g.Config.EID, gen))
		return g.allEffs(trades, surplusDays)
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

func (g *GA) allEffs(trades map[int][]tradeLPs, maxSurplus float64) []float64 {
	scores := make([]float64, g.N)
	for k, v := range trades {
		scores[k] = g.efficiency(v, maxSurplus)
	}
	return scores
}
// maxSurplus should be the max surplus after all the trading days and not per dat
func (g *GA) efficiency(trades []tradeLPs, maxSurplus float64) float64 {
	if len(trades) == 0{
		return 0.0
	}
	sum := 0.0
	for _, v := range trades {
		sum += v.TP - v.Slp + v.Blp - v.TP
	}
	eff := sum / maxSurplus

	log.WithFields(log.Fields{
		"trades": len(trades),
	}).Debug("Efficency: ", eff)
	return eff
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

func (g *GA) getLimitPrices(folderPath string) map[int][]tradeLPs {
	allTrades := make(map[int][]tradeLPs)
	for i := 0; i < g.N; i++ {
		fileName, err := filepath.Abs(folderPath + "IND_" + strconv.Itoa(i) + "/ExecOrders.csv")
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

		lines = lines[1:][:]
		// maps trading day to trade id to trade LPS
		tlps := make(map[int]map[int]tradeLPs)
		for _, line := range lines {
			day, _ := strconv.Atoi(line[0])
			ts, _ := strconv.Atoi(line[1])
			tid, _ := strconv.Atoi(line[3])
			lp, _ := strconv.ParseFloat(line[4], 64)
			tp, _ := strconv.ParseFloat(line[5], 64)
			otype := line[6]

			if tid == -1.0 {
				continue
			}

			if _, ok := tlps[day]; ok {
				if val, ok := tlps[day][tid]; ok{
					if otype == "BID"{
						val.Blp = lp
						tlps[day][tid] = val
					} else {
						val.Slp = lp
						tlps[day][tid] = val
					}
				} else {
					if otype == "BID"{
						tlps[day][tid] = tradeLPs{
							TID: tid,
							TS: ts,
							TD: day,
							TP: tp,
							Blp: lp,
						}
					} else {
						tlps[day][tid] = tradeLPs{
							TID: tid,
							TS: ts,
							TD: day,
							TP: tp,
							Slp: lp,
						}
					}
				}
			} else {
				tlps[day] = make(map[int]tradeLPs)
				if otype == "BID"{
					tlps[day][tid] = tradeLPs{
						TID: tid,
						TS: ts,
						TD: day,
						TP: tp,
						Blp: lp,
					}
				} else {
					tlps[day][tid] = tradeLPs{
						TID: tid,
						TS: ts,
						TD: day,
						TP: tp,
						Slp: lp,
					}
				}
			}
		}

		// Flatten map
		trades := []tradeLPs{}
		for _, v := range tlps {
			for _, t := range v {
				trades = append(trades, t)
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
			"WindowSizeEE",
			"DeltaEE",
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
			strconv.Itoa(v.WindowSizeEE),
			fmt.Sprintf("%.5f", v.DeltaEE),
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
			WindowSizeEE: 1,
			DeltaEE: 0.0,
			Dominance:    0,
			OrderQueuing: 1,
		}
	case "NORMAL":
		return exchange.AuctionParameters{
			BidAskRatio:  1,
			KPricing:     0.5,
			MinIncrement: 1,
			MaxShift:     2,
			WindowSizeEE: 3,
			DeltaEE: 5.0,
			Dominance:    0,
			OrderQueuing: 1,
		}
	case "HIGH":
		return exchange.AuctionParameters{
			BidAskRatio:  10,
			KPricing:     1,
			MinIncrement: 10,
			MaxShift:     100,
			WindowSizeEE: 5,
			DeltaEE: 20.0,
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
			WindowSizeEE: rand.Intn(5-1) + 1,
			DeltaEE: 2 + 10 * rand.Float64(),
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
			WindowSizeEE: rand.Intn(5-1) + 1,
			DeltaEE: 2 + 10 * rand.Float64(),
			Dominance:    rand.Intn(5),
			OrderQueuing: 1,
		}
	}
}

// calculates equilibrium price and equilibrium quantity
// the maximal theoretical number of trades is equal to the equilibrium quantity floored
// as no fractiontrade can be made
func calculateEQ(sps, bps []float64) (float64, int, error){
	sort.Float64s(sps)
	sort.Sort(sort.Reverse(sort.Float64Slice(bps)))
	//log.Warn("Sps: ",sps)
	//log.Warn("Bps: ", bps)
	 for ix, value := range sps {
	 	if len(bps) <= ix +1 {
	 		break
		}

		if bps[ix]  == value {
			return value, ix, nil
		} else if  bps[ix] < value {
			return (bps[ix] + value) / 2.0, ix, nil
		}
	 }

	 return -1.0, -1.0, errors.New("No intersection")
}

// Calculate max surplus fro sellers and buyers given the equilibrium price pe
func calculateMaxSurplus(sps, bps []float64, pe float64) (float64, float64) {
	sMaxSurplus := 0.0
	bMaxSurplus := 0.0

	for _, v := range sps {
		if v < pe {
			sMaxSurplus += pe - v
		}
	}

	for _, v := range bps {
		if v > pe {
			sMaxSurplus += v - pe
		}
	}

	return sMaxSurplus, bMaxSurplus
}