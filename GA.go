package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"mexs/exchange"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
	"mexs/bots"
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
	// Range // [0, 1]
	MutationRate float64
}

// TODO: change mutation rate

func (g *GA) Start() {
	rand.Seed(time.Now().UTC().UnixNano())
	log.WithFields(log.Fields{
		"EID":         g.Config.EID,
		"Individuals": g.N,
		"Gens":        g.Gens,
		"Fitness FN": g.Config.FitnessFN,
		"Chrmozone Init": g.Config.CInit,
	}).Warn("STARTING GA")
	// This function will be the heart of the GA
	// Number of individuals in each generation
	err := os.MkdirAll("../mexs/logs/"+g.Config.EID+"/", 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Log Folder for this experiment could not be made")
	}
	// Things that remain constant per generation

	// START BY initializing the chromosomes
	cs := make([]exchange.AuctionParameters, g.N)
	for i := 0; i < g.N; i++ {

		cs[i] = InitializeChromozones(g.Config.CInit)
	}
	g.currentGenes = cs

	low := false
	if g.Config.FitnessFN == "ALPHA" {
		low = true
	}

	for i := 0; i < g.Gens; i++ {
		log.Warn("GEN:", i)

		err := os.MkdirAll("../mexs/logs/"+g.Config.EID+"/GEN_"+strconv.Itoa(i)+"/", 0755)
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
			}).Error("Log Folder for this generation could not be made")
		}
		// Runs the current generation of markets
		g.MakeGen(g.currentGenes, strconv.Itoa(i))
		// Calculate score of each individual in the generation
		scores := g.FitnessFunction(g.Config.FitnessFN, strconv.Itoa(i))
		// Store the score of each individual in the generation
		g.chromozonesToCSV(i, g.currentGenes, scores)
		// Find best individual
		best, score, index := g.elitism(low, scores)
		g.logElite(best, score, index, strconv.Itoa(i))
		log.WithFields(log.Fields{
			"gens": best,
		}).Debug("Best Individual score: ", score)
		// Create new generation based on the scores of the previous one
		g.createNewGen(scores, low)
		// This passes the best individual unchanged from one generation to the next
		g.currentGenes[index] = best
	}

}
func (g *GA) createNewGen(scores []float64, low bool) {
	for i := 0; i < g.N; i++ {
		g.currentGenes[i] = g.getChildGenes(scores,low)
	}
}

// scores :- scores[i] is the score of the ith individual
// IF low == true then it means lower scores are better
func (g *GA) getChildGenes(scores []float64, low bool) exchange.AuctionParameters {

	contenders := []int{rand.Intn(g.N), rand.Intn(g.N), rand.Intn(g.N)}
	// ix1 -> MOM
	ix1 := 0
	// ix2 -> DAD
	ix2 := 0
	maxSc := scores[contenders[ix1]]
	if low{
		for i := 1; i < len(contenders); i++ {
			if scores[contenders[i]] < maxSc {
				maxSc = scores[contenders[i]]
				ix2 = ix1
				ix1 = i
			}
		}
	} else {
		for i := 1; i < len(contenders); i++ {
			if scores[contenders[i]] > maxSc {
				maxSc = scores[contenders[i]]
				ix2 = ix1
				ix1 = i
			}
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
	// KPricing mutation is in range of [-0.01, 0.01] with limits [0,1]
	kp = mutateFloat(mom.KPricing, dad.KPricing, 0.0, 1.0, 2, -1, 50, 3, 5, g.MutationRate)

	// MinIncrement mutation is in range of [-0.5, 05] with limit [0, 20]
	minI = mutateFloat(mom.KPricing, dad.KPricing, 0.0, 1.0, 5, -5, 50, 3, 4, g.MutationRate)

	// WindowSizeEE mutation is in range of [-1, +1] with limit [1, 20]
	win = mutateInt(mom.WindowSizeEE, dad.WindowSizeEE, 2, -1, 50, 1, 20, g.MutationRate)

	// DeltaEE mutation is in range of [-1.0, +1] with limit [0, 100]
	delta = mutateFloat(mom.DeltaEE, dad.DeltaEE, 0.0, 100.0, 2, -1, 50, 3, 3, g.MutationRate)

	// MaxShift mutation is in range of [-0.01, 0.01] with limit [0.05, 10]
	maxS = mutateFloat(mom.MaxShift, dad.MaxShift, 0.05, 10, 2, -1, 50, 3, 5, g.MutationRate)

	// Dominance mutation is in range of [-1, 1] with limit [0, 10]
	dom = mutateInt(mom.Dominance, dad.Dominance, 2, -1, 50, 0, 10, g.MutationRate)

	// BidAsk ratio mutation always mom gene  range [ -0.05, 0.05] with limit [0.2, 5]
	bar := mutateFloat(mom.BidAskRatio, mom.BidAskRatio, 0.2, 5.0, 6, -5, 0, 3, 5, g.MutationRate)
	return exchange.AuctionParameters{
		BidAskRatio:  bar,
		KPricing:     kp,
		MinIncrement: minI,
		WindowSizeEE: win,
		DeltaEE:      delta,
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
func mutateInt(mom, dad, max, min, prob, lbound, ubound int, mRate float64) int {
	mutation := 0
	if mRate > rand.Float64() {
		mutation = rand.Intn(max-min) + min
	}
	v := dad + mutation
	if val := rand.Intn(100); val < prob {
		v = mom + mutation
	}

	if v < lbound {
		return lbound
	} else if v > ubound {
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
func mutateFloat(mom, dad, lbound, ubound float64, max, min, prob, d1, d2 int, mRate float64) float64 {
	mutation := 0.0
	if mRate > rand.Float64() {
		nMax := max * int(math.Pow(10.0, float64(d1)))
		nMin := min * int(math.Pow(10.0, float64(d1)))
		randN := float64(rand.Intn(nMax-nMin) + nMin)
		mutation = randN / math.Pow(10.0, float64(d2))
	}

	v := dad + mutation
	if val := rand.Intn(100); val < prob {
		v = mom + mutation
	}

	if v < lbound {
		return lbound
	} else if v > ubound {
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
		ex.SetTraders(g.ReMakeAgents())
		ex.StartMarket(g.Config.EID+"/GEN_"+gen+"/IND_"+strconv.Itoa(i), g.Config.Schedule, g.Config.SandDs)
	}
}

func (g *GA) FitnessFunction(fnName string, gen string) []float64 {
	// Allow for different functions to be used
	switch fnName {
	case "ALPHA":
		// alpha
		trades := g.readTradesCSV(fmt.Sprintf("../mexs/logs/%s/GEN_%s/", g.Config.EID, gen))
		return g.allAlphaScores(trades)

	case "ALOC-EFF":
		// FIXME: assume no market shocks for now
		// Regular schedule
		pe, _, err := calculateEQ(g.Config.Sps, g.Config.Bps)
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err.Error(),
				"pe":    pe,
			}).Panic("Failed to calculates EQ in fitness function")
		}
		sS, bS := calculateMaxSurplus(g.Config.Sps, g.Config.Bps, pe)
		surplusDays := (sS + bS) * float64(g.Config.MarketInfo.TradingDays)
		trades := g.getLimitPrices(fmt.Sprintf("../mexs/logs/%s/GEN_%s/", g.Config.EID, gen))
		log.WithFields(log.Fields{
			"surplusDays": surplusDays,
			"Pe":          pe,
			"trades": trades,
		}).Debug("Efficency data")

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
	return alpha
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
	if len(trades) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range trades {
		// Seller profit is  =  Trade price  - Seller limit price
		// buyers profit is  = Buyer limit price  - Trade Price
		// Total profit is = buyer profit + seller profit
		sum += (v.TP - v.Slp) + (v.Blp - v.TP)
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

// return trades and limit prices  in form map[individual]structure
func (g *GA) getLimitPrices(folderPath string) map[int][]tradeLPs {
	allTrades := make(map[int][]tradeLPs)
	for i:=0; i< g.N; i++ {
		// READ trades.csv
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

		// remove headers from csv
		lines = lines[1:][:]
		log.Debug(lines)
		trades := make([]tradeLPs, len(lines))
		for ix, line := range lines {
			id, _ := strconv.Atoi(line[0])
			td, _ := strconv.Atoi(line[1])
			ts, _ := strconv.Atoi(line[2])
			p, _ := strconv.ParseFloat(line[3], 64)

			ap, _ := strconv.ParseFloat(line[8], 64)
			bp, _ := strconv.ParseFloat(line[9], 64)

			trades[ix] = tradeLPs{
				TID:  id,
				TD:  td,
				TS:  ts,
				TP:   p,
				Slp: ap,
				Blp: bp,
			}
		}
		allTrades[i] = trades
		file.Close()
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

	if len(scores) !=  len(cs) {
		log.WithFields(log.Fields{
			"Scores len ": len(scores),
			"Len cs": len(cs),
		}).Panic("Size of chromosomes array does not match score array")
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
			DeltaEE:      0.0,
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
			DeltaEE:      5.0,
			Dominance:    0,
			OrderQueuing: 1,
		}
	case "HIGH":
		return exchange.AuctionParameters{
			BidAskRatio:  5,
			KPricing:     1,
			MinIncrement: 10,
			MaxShift:     10,
			WindowSizeEE: 10,
			DeltaEE:      20.0,
			Dominance:    10,
			OrderQueuing: 1,
		}
	case "RANDOM":
		// random, in logical range
		// BidAsk Ratio between [0.7, 1.2)
		// KPricing between [0,1)
		// MinIncrement between [0,5)
		// MaxShift between [0.5,1.5)
		// Domminance between [0,5)
		return exchange.AuctionParameters{
			BidAskRatio:  float64(rand.Intn(13-7)+7) / 10.0,
			KPricing:     rand.Float64(),
			MinIncrement: float64(rand.Intn(5)),
			MaxShift:     rand.Float64() + 0.5,
			WindowSizeEE: rand.Intn(6-1) + 1,
			DeltaEE:      2 + 5*rand.Float64(),
			Dominance:    rand.Intn(5),
			OrderQueuing: 1,
		}
	default:
		// Note: default to any especial case I want to test out
		return exchange.AuctionParameters{
			BidAskRatio:  float64(rand.Intn(13-7)+7) / 10.0,
			KPricing:     rand.Float64(),
			MinIncrement: float64(rand.Intn(5)),
			MaxShift:     rand.Float64() + 0.5,
			WindowSizeEE: rand.Intn(6-1) + 1,
			DeltaEE:      2 + 5*rand.Float64(),
			Dominance:    rand.Intn(5),
			OrderQueuing: 1,
		}
	}
}

// calculates equilibrium price and equilibrium quantity
// the maximal theoretical number of trades is equal to the equilibrium quantity floored
// as no fraction trade can be made
func calculateEQ(sps, bps []float64) (float64, int, error) {
	sort.Float64s(sps)
	sort.Sort(sort.Reverse(sort.Float64Slice(bps)))
	//log.Warn("Sps: ",sps)
	//log.Warn("Bps: ", bps)
	for ix, value := range sps {
		if len(bps) <= ix+1 {
			break
		}

		if bps[ix] == value {
			return value, ix, nil
		} else if bps[ix] < value {
			return (bps[ix] + value) / 2.0, ix, nil
		}
	}

	return -1.0, -1.0, errors.New("No intersection")
}

// Function is used to have elitism in the GA
// This is the practice by which the best individual of each
// generation is past un altered to the next gen
func (g *GA) elitism(low bool, scores []float64) (exchange.AuctionParameters, float64, int) {
	// low == True means that low scores are better
	bestScore := scores[0]
	bix := 0
	if low {
		for i := 1; i < len(scores); i++ {
			if scores[i] < bestScore {
				bix = i
				bestScore = scores[i]
			}
		}
	} else {
		for i := 1; i < len(scores); i++ {
			if scores[i] > bestScore {
				bestScore = scores[i]
				bix = i
			}
		}
	}

	return g.currentGenes[bix], bestScore, bix
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

func (g *GA) logElite(elite exchange.AuctionParameters, score float64, ix int, gen string) {
	// TODO: Save the elite of each generation to CSV file for later study
	// of the evolution of the elites thought the evolution process
	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/elite.csv", g.Config.EID))
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
			"ID",
			"B:A",
			"K",
			"MinIncrement",
			"WindowSizeEE",
			"DeltaEE",
			"MaxShift",
			"Dominance",
		})
	}
	writer.Write([]string{
		gen,
		fmt.Sprintf("%.4f", score),
		strconv.Itoa(ix),
		fmt.Sprintf("%.5f", elite.BidAskRatio),
		fmt.Sprintf("%.5f", elite.KPricing),
		fmt.Sprintf("%.5f", elite.MinIncrement),
		strconv.Itoa(elite.WindowSizeEE),
		fmt.Sprintf("%.5f", elite.DeltaEE),
		fmt.Sprintf("%.5f", elite.MaxShift),
		strconv.Itoa(elite.Dominance),
	})

}

func (g* GA) ReMakeAgents() map[int]bots.RobotTrader {
	traders := make(map[int]bots.RobotTrader)
	for i, id := range g.Config.SellersIDs {
		switch g.Config.AlgoS[i] {
		case "ZIP":
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "SELLER", g.Config.MarketInfo)
			traders[zipT.Info.TraderID] = zipT
		case "ZIC":
			zic := &bots.ZICTrader{}
			zic.InitRobotCore(id, "SELLER", g.Config.MarketInfo)
			traders[zic.Info.TraderID] = zic
		case "AA":
			aa := &bots.AATrader{}
			aa.InitRobotCore(id, "SELLER", g.Config.MarketInfo)
			traders[aa.Info.TraderID] = aa
		default:
			log.Panic("SHIIT")
		}
	}

	for i, id := range g.Config.BuyersIDs {
		switch g.Config.AlgoB[i] {
		case "ZIP":
			zipT := &bots.ZIPTrader{}
			zipT.InitRobotCore(id, "BUYER", g.Config.MarketInfo)
			traders[zipT.Info.TraderID] = zipT
		case "ZIC":
			zic := &bots.ZICTrader{}
			zic.InitRobotCore(id, "BUYER", g.Config.MarketInfo)
			traders[zic.Info.TraderID] = zic
		case "AA":
			aa := &bots.AATrader{}
			aa.InitRobotCore(id, "BUYER", g.Config.MarketInfo)
			traders[aa.Info.TraderID] = aa
		default:
			log.Panic("SHIIT")
		}
	}
	return traders
}