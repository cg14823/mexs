package bots

import (
	"mexs/common"
	"math/rand"
	"errors"
	"os"
	"encoding/csv"
	"strconv"
	"fmt"
	"path/filepath"
	log "github.com/sirupsen/logrus"

	"math"
	"time"
)

// Most of the code in this file is a port of Dave cliff
//implementation of AA that can be found on BristolStockEchange
type AATrader struct {
	Info RobotCore
	//External parameters
	spinUpTime int
	eta float64
	thetaMax float64
	thetaMin float64
	lambdaA float64
	lambdaR float64
	beta1 float64
	beta2 float64
	gamma float64
	nLastTrades int
	ema float64
	maxNewtonItter int
	maxNewtonError float64

	// Internal params
	equilibrium float64
	theta float64
	smithsAlpha float64
	smithsAlphaMin float64
	smithsAlphaMax float64

	agresBuy float64
	agresSell float64
	targetBuy float64
	targetSell float64
	target float64

	// current job details
	limitPrice float64
	active bool
	job *TraderOrder

	// Parameters describing market
	prevBestBid float64
	prevBestAsk float64
	lastTrades []float64
}

func (t *AATrader) InitRobotCore(id int, sellerOrBuyer string, marketInfo common.MarketInfo) {
	t.Info = RobotCore{
		TraderID: id,
		Type: "AA",
		SellerOrBuyer: sellerOrBuyer,
		ExecutionOrders: []*TraderOrder{},
		MarketInfo: marketInfo,
		ActiveOrders: map[int]*common.Order{},
		Balance: 0,
	}

	t.spinUpTime = 20
	t.eta = 3.0
	t.thetaMax = 2.0
	t.thetaMin = 8.0
	t.lambdaA = 0.01
	t.lambdaR = 0.02
	t.beta1 = 0.4
	t.beta2 = 0.4
	t.gamma = 2.0
	t.nLastTrades = 5
	t.ema = 2.0 / (float64(t.nLastTrades + 1))
	t.maxNewtonItter = 10
	t.maxNewtonError = 0.0001

	t.active = false

	t.theta = -1.0 * (5.0 * rand.Float64())

	t.agresBuy = -1.0 * 0.3 * rand.Float64()
	t.agresSell = -1.0 * 0.3 * rand.Float64()
	t.lastTrades = []float64{}

	// Uninitialized values
	t.prevBestAsk = -1.0
	t.prevBestBid = -1.0
	t.equilibrium = -1.0
	t.smithsAlphaMin = -1.0
	t.smithsAlphaMax = -1.0
	t.smithsAlpha = -1.0
	t.targetBuy = -1.0
	t.targetSell = -1.0
}

func (t *AATrader) SetOrders(orders []*TraderOrder) {
	t.Info.ExecutionOrders = orders
}

func (t *AATrader) AddOrder(order *TraderOrder) {
	t.Info.ExecutionOrders = append(t.Info.ExecutionOrders, order)
}

func (t *AATrader) RemoveOrder() error {
	if len(t.Info.ExecutionOrders) == 0 {
		return errors.New("no order to be removed")
	}

	t.Info.ExecutionOrders = t.Info.ExecutionOrders[:len(t.Info.ExecutionOrders)-1]
	if len(t.Info.ExecutionOrders) != 0 {
		t.limitPrice = t.Info.ExecutionOrders[0].LimitPrice
	}
	return nil
}

func (t *AATrader) LogBalance(fileName string, day int, trade *common.Trade) {
	fileName, err := filepath.Abs(fileName + "/AATradersLog.csv")
	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day": day,
			"error":       err.Error(),
		}).Error("File Path not found")
		return
	}

	addHeader := true
	if _, err := os.Stat(fileName); err == nil {
		addHeader = false
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()

	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day": day,
			"error":       err.Error(),
		}).Error("AA trader CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if addHeader {
		writer.Write([]string{"Day", "TimeStep", "TID", "TradeID", "Profit", "TPrice"})
	}

	writer.Write([]string{
		strconv.Itoa(day),
		strconv.Itoa(trade.TimeStep),
		strconv.Itoa(t.Info.TraderID),
		strconv.Itoa(trade.TradeID),
		fmt.Sprintf("%.5f", t.Info.Balance),
		fmt.Sprintf("%.5f", trade.Price),
	})
}

func (t *AATrader) LogOrder(fileName string, d, ts, tradeID int, tPrice float64) {
	if len(t.Info.ExecutionOrders) == 0 {
		log.Warn("Log order called with no orders")
		return
	}
	// For now assume agents have only one order at a time
	fileName, err := filepath.Abs(fileName)
	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day": d,
			"error":       err.Error(),
		}).Error("File Path not found")
		return
	}
	addHeader := true
	if _, err := os.Stat(fileName); err == nil {
		addHeader = false
	}

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()

	if err != nil {
		log.WithFields(log.Fields{
			"Trading Day": d,
			"error":       err.Error(),
		}).Error("AA exec order CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if addHeader {
		writer.Write([]string{"Day", "TimeStep", "TID", "TradeID", "LimitPrice", "TPrice", "OType", "Algo"})
	}

	writer.Write([]string{
		strconv.Itoa(d),
		strconv.Itoa(ts),
		strconv.Itoa(t.Info.TraderID),
		strconv.Itoa(tradeID),
		fmt.Sprintf("%.5f", t.Info.ExecutionOrders[0].LimitPrice),
		fmt.Sprintf("%.5f", tPrice),
		t.Info.ExecutionOrders[0].Type,
		"AA",
	})
}

func (t *AATrader) GetExecutionOrder() []*TraderOrder {
	return t.Info.ExecutionOrders
}

func (t *AATrader) GetOrder(timeStep int) *common.Order {
	if len(t.Info.ExecutionOrders) == 0 {
		t.active = false
		t.job = &TraderOrder{Type: "NA"}
		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NA",
		}
	}

	t.active = true
	t.job = t.Info.ExecutionOrders[0]
	t.limitPrice = t.job.LimitPrice
	t.updateTarget()

	var quotePrice float64
	 // TODO: change target Buy target sell to target??
	if t.job.IsBid() {
		if t.spinUpTime > 0 {
			askPlus := (1 + t.lambdaR) * t.prevBestAsk +t.lambdaA
			quotePrice = t.prevBestBid +
				(math.Min(t.limitPrice, askPlus) - t.prevBestBid) / t.eta
		} else {
			quotePrice = t.prevBestBid + (t.targetBuy - t.prevBestBid) / t.eta
		}
	} else {
		if t.spinUpTime > 0 {
			bidMinus := (1 - t.lambdaR) * t.prevBestBid - t.lambdaA
			quotePrice = t.prevBestAsk -
				(t.prevBestAsk - math.Max(t.limitPrice, bidMinus)) / t.eta
		} else {
			quotePrice = t.prevBestAsk - (t.prevBestAsk - t.targetSell) / t.eta
		}
	}

	return &common.Order{
		TraderID: t.Info.TraderID,
		OrderType: t.job.Type,
		Price: quotePrice,
		Quantity: t.job.Quantity,
		TimeStep: timeStep,
		Time: time.Now(),
	}
}

func (t *AATrader) 	MarketUpdate(update common.MarketUpdate) {
	// bid LOB
	bidImproved := false
	bidHit := false
	if update.BestBid != -1 {
		if t.prevBestBid < update.BestBid || t.prevBestBid == -1 {
			bidImproved = true
		} else if update.LastTrade.TimeStep == update.TimeStep &&
			(t.prevBestBid >= update.BestBid || t.prevBestBid == -1) {
				bidHit = true
		}
	} else if t.prevBestBid != -1 {
		bidHit = true
	}

	// Ask LOB
	askImproved := false
	askLifted := false
	if update.BestAsk != -1 {
		if t.prevBestAsk > update.BestAsk || t.prevBestAsk == -1 {
			askImproved = true
		} else if update.LastTrade.TimeStep == update.TimeStep &&
			(t.prevBestAsk <= update.BestAsk || t.prevBestAsk == -1) {
			askLifted = true
		}
	} else if t.prevBestAsk != -1 {
		askLifted = true
	}

	deal := bidHit || askLifted
	t.prevBestAsk = update.BestAsk
	t.prevBestBid = update.BestBid

	if t.spinUpTime > 0 {
		t.spinUpTime --
	}

	if deal {
		price := update.LastTrade.Price
		t.updateEq(price)
		t.updateSmithsAlpha(price)
		t.updateTheta()

		// For buying
		if t.targetBuy >= price {
			t.agresBuy = t.updateAgg(false, true, price)
		} else {
			t.agresBuy = t.updateAgg(true, true, price)
		}

		// For selling
		if t.targetSell <= price {
			t.agresSell = t.updateAgg(false, false, price)
		} else {
			t.agresSell = t.updateAgg(true, false, price)
		}
	} else {
		if bidImproved && t.targetBuy <= t.prevBestBid {
			t.agresBuy = t.updateAgg(true, true, t.prevBestBid)
		}
		if askImproved && t.targetSell >= t.prevBestAsk {
			t.agresSell = t.updateAgg(true, false, t.prevBestAsk)
		}
	}

	t.updateTarget()
}

func (t *AATrader) TradeMade(trade *common.Trade) bool {
	t.Info.TradeRecord = append(t.Info.TradeRecord, trade)
	if trade.SellOrder.TraderID == t.Info.TraderID {
		t.Info.Balance += trade.Price - t.Info.ExecutionOrders[0].LimitPrice
	} else {
		t.Info.Balance += t.Info.ExecutionOrders[0].LimitPrice - trade.Price
	}

	t.RemoveOrder()
	if len(t.Info.ExecutionOrders) == 0 {
		t.active = false
	}

	return true
}

// AA Helper functions
func (t AATrader) calcRShout(target float64, buying bool) float64 {
	if buying {
		// Extramarginal
		if t.equilibrium >= t.limitPrice{
			return 0.0
		}
		if target > t.equilibrium {
			newTarget := target
			if target > t.limitPrice {
				newTarget = t.limitPrice
			}
			rShout := math.Log((((newTarget - t.equilibrium) *
				(math.Exp(t.theta) -1)) / (t.limitPrice - t.equilibrium)) + 1) / t.theta
			return  rShout
		}
		rShout := math.Log((1 - (target / t.equilibrium)) *
			(math.Exp(t.newton4Buying()) -1 ) + 1) / (- t.newton4Buying())
		return  rShout
	}

	// selling
	if t.limitPrice >= t.equilibrium {
		return 0.0
	}
	if target > t.equilibrium {
		rShout := math.Log(((target - t.equilibrium) *
			(math.Exp(t.newton4Selling()) - 1)) /
				(t.Info.MarketInfo.MaxPrice - t.equilibrium) + 1) / (-t.newton4Selling())
		return  rShout
	}
	newTarget := target
	if target < t.limitPrice {
		newTarget = t.limitPrice
	}
	rShout := math.Log((1 - (newTarget -t.limitPrice) / (t.equilibrium - t.limitPrice)) *
		(math.Exp(t.theta) -1) + 1) / t.theta
	return  rShout
}

func (t *AATrader) updateAgg(up , buying bool, target float64) float64 {
	oldAgg := t.agresSell
	if buying {
		oldAgg = t.agresBuy
	}

	var delta float64
	if up {
		delta = (1 + t.lambdaR) * t.calcRShout(target, buying) + t.lambdaA
	} else {
		delta = (1 - t.lambdaR) * t.calcRShout(target, buying) - t.lambdaA
	}

	newAgg := oldAgg + t.beta1 * (delta -oldAgg)
	if newAgg > 1.0 {
		newAgg = 0
	} else if newAgg < 0.0 {
		newAgg = 0.000001
	}
	return newAgg
}

func (t *AATrader) updateTheta() {
	alphaBar := (t.smithsAlpha - t.smithsAlphaMin) / (t.smithsAlphaMax - t.smithsAlphaMin)
	desiredTheta := (t.thetaMax -t.thetaMin) * (1 -(alphaBar *
		math.Exp(t.gamma * (alphaBar - 1)))) + t.thetaMin
	theta := t.theta + t.beta2 * (desiredTheta - t.theta)
	if theta == 0.0 {
		theta += 0.0000001
	}
	t.theta = theta
}

func (t * AATrader) updateSmithsAlpha(price float64){
	t.lastTrades = append(t.lastTrades, price)
	if !(len(t.lastTrades) <= t.nLastTrades) {
		t.lastTrades = t.lastTrades[1:]
	}
	sum := 0.0
	for _, v := range t.lastTrades {
		sum += math.Pow(v - t.equilibrium,2)
	}
	t.smithsAlpha = math.Sqrt(sum * (1 / float64(len(t.lastTrades)))) / t.equilibrium

	if t.smithsAlphaMin == -1.0 {
		t.smithsAlphaMin = t.smithsAlpha
		t.smithsAlphaMax = t.smithsAlpha
	} else {
		if t.smithsAlpha < t.smithsAlphaMin {
			t.smithsAlphaMin = t.smithsAlpha
		}
		if t.smithsAlpha > t.smithsAlphaMax {
			t.smithsAlphaMax = t.smithsAlpha
		}
	}
}

func (t *AATrader) updateEq(price float64) {
	if t.equilibrium == -1 {
		t.equilibrium = price
	} else {
		t.equilibrium = t.ema * price + (1 - t.ema) * t.equilibrium
	}
}

func (t *AATrader) updateTarget() {
	// For buying
	if t.limitPrice < t.equilibrium {
		// Extra-marginal buyer
		if t.agresBuy >= 0{
			t.targetBuy = t.limitPrice
		} else {
			t.targetBuy = t.limitPrice *
				(1 - (math.Exp(-t.agresBuy * t.theta) -1 )) /
				(math.Exp(t.theta) - 1.0)
		}
	} else {
		// Intra-marginal buyer
		if t.agresBuy >=0 {
			t.targetBuy = t.equilibrium + (t.limitPrice - t.equilibrium) *
				((math.Exp(t.agresBuy * t.theta) - 1) / (math.Exp(t.theta) - 1))
		} else {
			thetaEst := t.newton4Buying()
			t.targetBuy = t.equilibrium *
				(1 - (math.Exp(-t.agresBuy * thetaEst) - 1) / (math.Exp(thetaEst) - 1))
		}
	}

	// For Selling
	if t.limitPrice > t.equilibrium {
		if t.agresSell >= 0 {
			t.targetSell = t.limitPrice
		} else {
			t.targetSell = t.limitPrice +
				(t.Info.MarketInfo.MaxPrice - t.equilibrium) *
					((math.Exp(-t.agresSell * t.theta) - 1) / (math.Exp(t.theta) - 1))
		}
	} else {
		// Intra-marginal seller
		if t.agresSell >= 0 {
			t.targetSell = t.limitPrice + (t.equilibrium -t.limitPrice) *
				(1 - (math.Exp(t.agresSell * t.theta) -1) / (math.Exp(t.theta) -1 ))
		} else {
			thetaEst := t.newton4Selling()
			t.targetSell = t.equilibrium +
				(t.Info.MarketInfo.MaxPrice - t.equilibrium) *
					((math.Exp(-t.agresSell * thetaEst) -1) / (math.Exp(thetaEst) - 1))
		}
	}
}

func (t *AATrader) newton4Buying() float64 {
	thetaEst := t.theta
	rightHSide := (t.theta * (t.limitPrice - t.equilibrium)) / (math.Exp(t.theta) -1)

	for i := 0; i <= t.maxNewtonItter ; i++ {
		eX := math.Exp(thetaEst)
		exMinOne := eX - 1
		fofX := ((thetaEst * t.equilibrium) / exMinOne) - rightHSide
		if math.Abs(fofX) <= t.maxNewtonError {
			break
		}
		dfofx := (t.equilibrium / exMinOne) -
			(eX * t.equilibrium * thetaEst) / (exMinOne * exMinOne)
		thetaEst = thetaEst - (fofX / dfofx)
	}
	if thetaEst == 0.0 {
		return 0.000001
	}
	return thetaEst
}

func (t *AATrader) newton4Selling() float64 {
	thetaEst := t.theta
	rightHSide := (t.theta * (t.equilibrium - t.limitPrice)) / (math.Exp(t.theta) - 1)
	for i:=0; i <= t.maxNewtonItter; i++{
		eX := math.Exp(thetaEst)
		exMinOne := eX - 1
		fofX := ((thetaEst * (t.Info.MarketInfo.MaxPrice - t.equilibrium)) /
			math.Exp(exMinOne)) - rightHSide
		if math.Abs(fofX) <= t.maxNewtonError {
			break
		}

		dfofx := ((t.Info.MarketInfo.MaxPrice - t.equilibrium)/ exMinOne) -
			((eX * (t.Info.MarketInfo.MaxPrice - t.equilibrium) * thetaEst) /
				(exMinOne * exMinOne))
		thetaEst = thetaEst - (fofX / dfofx)
	}

	if thetaEst == 0.0 {
		return 0.000001
	}

	return thetaEst
}

var _ RobotTrader = (*AATrader)(nil)