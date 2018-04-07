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

type schedData struct {
	SID int
	EqP float64
	EqQ int
	bSurplus float64
	sSurplus float64
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
	// Stats fro schedule used in day d where d is key for the map
	EqSched map[int]schedData
	// Seller limit prices in schedule s
	Sps map[int][]float64
	// buyer limit prices in schedule s
	Bps map[int][]float64
}

func (g *GA) Start() {
	// This function will be the heart of the GA
	rand.Seed(time.Now().UTC().UnixNano())
	// calculate equilibrium and other stats for schedules
	g.Sps, g.Bps = getLimits(g.Config)
	var errorEQ error
	g.EqSched, errorEQ = calculateAllEQ(g.Config.Schedule, g.Config.SandDs)
	if errorEQ != nil {
		log.Panic("Experiment can not be run with non intersecting supply and demand curves")
	}

	g.MutationRate = 0.25

	log.WithFields(log.Fields{
		"EID":         g.Config.EID,
		"Individuals": g.N,
		"Gens":        g.Gens,
		"Fitness FN": g.Config.FitnessFN,
		"Chromozone Init": g.Config.CInit,
		"EqSchde": g.EqSched,
		"Mutation rate": g.MutationRate,
	}).Warn("STARTING GA")

	// create folder that will contain logs
	err := os.MkdirAll("../mexs/logs/"+g.Config.EID+"/", 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Log Folder for this experiment could not be made")
	}

	// START BY initializing the genomes
	cs := make([]exchange.AuctionParameters, g.N)
	for i := 0; i < g.N; i++ {

		cs[i] = InitializeChromozones(g.Config.CInit)
	}
	g.currentGenes = cs

	// If fitness function is based on alpha then the smaller the better
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
		// decrease the mutation rate by 2 every 50 generations
		g.decayMutationRate(50, i, 2)
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
	// KPricing mutation is in range of [-0.05, 0.05] with limits [0,1]
	kp = mutateFloatSimple(mom.KPricing, dad.KPricing, 0.0, 1.0, g.MutationRate, 0.5, 0.05. -0.05)

	// MinIncrement mutation is in range of [-0.5, 0.5] with limit [0, 20]
	minI = mom.MinIncrement

	// WindowSizeEE mutation is in range of [-1, +1] with limit [1, 20]
	win = mutateIntBy1(mom.WindowSizeEE, dad.WindowSizeEE, 1, 20, 50, g.MutationRate)

	// DeltaEE mutation is in range of [-1.0, +1] with limit [0, 100]
	delta = mutateFloatSimple(mom.DeltaEE, dad.DeltaEE, 0.0, 200.0, g.MutationRate, 0.5, 1.0, -1.0)

	// MaxShift mutation is in range of [-0.02, 0.02] with limit [0.05, 10]
	maxS = mutateFloatSimple(mom.MaxShift, dad.MaxShift, 0.05, 10, g.MutationRate, 0.5, 0.2, -0.2)

	// Dominance mutation is in range of [-1, 1] with limit [0, 10]
	dom = mutateIntBy1(mom.Dominance, dad.Dominance, 0, 10, 50, g.MutationRate)

	// BidAsk ratio mutation always mom gene  range [ -0.05, 0.05] with limit [0.1, 0.9]
	bar := mutateFloatSimple(mom.BidAskRatio, mom.BidAskRatio, 0.1, 0.9, g.MutationRate, 1.0, 0.05, 0.05 )
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


func mutateIntBy1(mom, dad, lbound,ubound, prob int, mRate float64) int {
	mutation := 0
	if mRate > rand.Float64() {
		mutation = 1
		if rand.Float64() <= 0.5 {
			mutation = -1
		}
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

//centered around 0
func mutateFloatSimple(mom, dad, lbound, ubound, mRate, prob, maxM, minM float64) (float64){
	mutation := 0.0
	if mRate > rand.Float64() {
		mutation = rand.Float64() * (maxM- minM) + minM
	}

	v := dad +mutation
	if rand.Float64() < prob {
		 v = mom + mutation
	}
	if v < lbound {
		return lbound
	} else if v > ubound {
		return ubound
	}

	return v
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

		trades := g.getLimitPrices(fmt.Sprintf("../mexs/logs/%s/GEN_%s/", g.Config.EID, gen))
		return g.allEffs(trades)
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
	//
	tNum := float64(len(trades))
	if tNum == 0 {
		// Big value to penalise exchanges that make no trades happen
		return 100
	}

	// There is one alpha per day as market shocks occur between days for now
	alphas := make([]float64, g.Config.MarketInfo.TradingDays)
	sums := make([]float64, g.Config.MarketInfo.TradingDays)


	for _, t := range trades {
		sums[t.TD] += math.Pow(t.P-g.EqSched[t.TD].EqP, 2.0)
		// use alphas to store count of trades per day, to save some memory
		alphas[t.TD] ++
	}

	// score si the average alpha between trading days
	alpha := 0.0
	for d:= 0; d < g.Config.MarketInfo.TradingDays; d++ {
		// Penalize market with no trades
		if alphas[d] == 0 {
			sums[d] = 100000
			alphas[d] = 1
		}

		sums[d] = math.Sqrt(sums[d]/alphas[d])
		alphas[d] = (100.0 / g.EqSched[d].EqP) * sums[d]
		alpha += alphas[d]
	}
	alpha = alpha / float64(g.Config.MarketInfo.TradingDays)
	return alpha
}

func (g *GA) allEffs(trades map[int][]tradeLPs) []float64 {
	scores := make([]float64, g.N)
	for k, v := range trades {
		scores[k] = g.efficiency(v)
	}
	return scores
}

// Average efficency between days
func (g *GA) efficiency(trades []tradeLPs) float64 {
	if len(trades) == 0 {
		return 0.0
	}

	effs := make([]float64, g.Config.MarketInfo.TradingDays)
	for _, v := range trades {
		// Seller profit is  =  Trade price  - Seller limit price
		// buyers profit is  = Buyer limit price  - Trade Price
		// Total profit is = buyer profit + seller profit
		effs[v.TD] += (v.TP - v.Slp) + (v.Blp - v.TP)
	}

	eff := 0.0
	for d:= 0; d < g.Config.MarketInfo.TradingDays; d++ {
		effs[d] = effs[d] / (g.EqSched[d].bSurplus + g.EqSched[d].sSurplus)
		eff += effs[d]
	}
	eff = eff / float64(g.Config.MarketInfo.TradingDays)

	log.WithFields(log.Fields{
		"trades": len(trades),
	}).Debug("Efficiency: ", eff)
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
			BidAskRatio:  0.25,
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
			BidAskRatio:  0.5,
			KPricing:     0.5,
			MinIncrement: 1,
			MaxShift:     2,
			WindowSizeEE: 10,
			DeltaEE:      10.0,
			Dominance:    0,
			OrderQueuing: 1,
		}
	case "HIGH":
		return exchange.AuctionParameters{
			BidAskRatio:  0.75,
			KPricing:     1,
			MinIncrement: 10,
			MaxShift:     10,
			WindowSizeEE: 50,
			DeltaEE:      100.0,
			Dominance:    10,
			OrderQueuing: 1,
		}
	case "RANDOM":
		// random, in logical range
		// BidAsk Ratio between [0.1, 0.9)
		// KPricing between [0,1)
		// MinIncrement between [0,5)
		// MaxShift between [0.5,1.5)
		// Domminance between [0,5)
		return exchange.AuctionParameters{
			BidAskRatio:  rand.Float64() * (0.9-0.1) + 0.1,
			KPricing:     rand.Float64(),
			MinIncrement: float64(rand.Intn(5)),
			MaxShift:     rand.Float64() + 0.5,
			WindowSizeEE: rand.Intn(50-10) + 10,
			DeltaEE:      8+ 5*rand.Float64(),
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

func (g *GA) logElite(elite exchange.AuctionParameters, score float64, ix int, gen string) {
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

func getLimits(c ExperimentConfig) (map[int][]float64, map[int][]float64){
	// Case 1: when there is only one s and d
	sps := make(map[int][]float64)
	bps := make(map[int][]float64)
	for _, s := range c.SandDs {
		var sPrices []float64
		var bPrices []float64
		for _, alp := range s.Sps {
			sPrices = append(sPrices, alp.Prices...)
		}

		for _, alp := range s.Bps {
			bPrices = append(bPrices, alp.Prices...)
		}

		sps[s.ID] = sPrices
		bps[s.ID] = bPrices
	}

	return sps, bps
}

// calculates equilibrium price and equilibrium quantity
// the maximal theoretical number of trades is equal to the equilibrium quantity floored
// as no fraction trade can be made
func calculateAllEQ(sched exchange.AllocationSchedule, SAndDs map[int]exchange.SandD) (map[int]schedData, error) {
	results := make(map[int]schedData)

	for d, _ := range sched.Schedule {
		for _ , sid := range sched.Schedule[d] {
			if _, ok := results[sid]; !ok {
				data, err := calculateSchedEQ(SAndDs[sid])
				if err != nil {
					log.Panic("The stats could not be calculated for schedule ", sid)
				}
				results[sid] = data
			}
		}
	}

	return results, nil
}

func calculateSchedEQ(s exchange.SandD) (schedData, error) {
	var sPrices []float64
	var bPrices []float64

	for _, alp := range s.Sps {
		sPrices = append(sPrices, alp.Prices...)
	}

	for _, alp := range s.Bps {
		bPrices = append(bPrices, alp.Prices...)
	}

	sort.Float64s(sPrices)
	sort.Sort(sort.Reverse(sort.Float64Slice(bPrices)))

	for ix, value := range sPrices {
		if len(bPrices) <= ix + 1 {
			break
		}

		if bPrices[ix] == value {
			eqP := value
			eqQ := ix
			sellerS, buyerS := calculateMaxSurplus(sPrices, bPrices, eqP)
			return schedData{
				SID: s.ID,
				EqP:eqP,
				EqQ:eqQ,
				sSurplus:sellerS,
				bSurplus:buyerS,
			}, nil
		} else if bPrices[ix] < value {
			eqP := (bPrices[ix] + value) / 2.0
			eqQ := ix
			sellerS, buyerS := calculateMaxSurplus(sPrices, bPrices, eqP)
			return schedData{
				SID: s.ID,
				EqP:eqP,
				EqQ:eqQ,
				sSurplus:sellerS,
				bSurplus:buyerS,
			}, nil
		}
	}

	return schedData{}, errors.New("No intersection")
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


// decrease mutation rate every X steps
func (g *GA) decayMutationRate(steps, gen int, factor float64) {
	if gen != 0 && gen % steps == 0 {
		g.MutationRate = g.MutationRate / factor
	}
}