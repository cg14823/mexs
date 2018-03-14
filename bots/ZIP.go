package bots

// This trader is based on Dave Cliff's  1997 work, most of the logic has been ported
// from Dave Cliff's open source project 'BristolStockExchange' available at
// https://github.com/davecliff/BristolStockExchange, parts of it where modified to
// better fit this simulator, but it still makes the same decisions overall.

import (
	"encoding/csv"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"mexs/common"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type ZIPTrader struct {
	Info             RobotCore
	job              *TraderOrder
	limitPrice       float64
	active           bool
	lastDelta        float64
	beta             float64
	momentum         float64
	margin           float64
	marginBuy        float64
	marginSell       float64
	ca               float64
	cr               float64
	price            float64
	prevBestBidPrice float64
	prevBestBidQty   int
	prevBestAskPrice float64
	prevBestAskQty   int
}

func (t *ZIPTrader) InitRobotCore(id int, sellerOrBuyer string, marketInfo common.MarketInfo) {
	t.Info = RobotCore{
		TraderID:        id,
		Type:            "ZIP",
		SellerOrBuyer:   sellerOrBuyer,
		ExecutionOrders: []*TraderOrder{},
		MarketInfo:      marketInfo,
		ActiveOrders:    map[int]*common.Order{},
		Balance:         0,
	}

	// Initialize ZIP parameters following Dave cliff 1997 paper procedure
	t.active = false
	t.lastDelta = 0.0
	t.beta = 0.1 + 0.4*rand.Float64()
	t.momentum = 0.2 + rand.Float64()*0.6
	t.ca = 0.05 // t.ca & .cr were hard-coded in '97 but parameterised later
	t.cr = 0.05
	t.marginBuy = -(0.05 + 0.3 * rand.Float64())
	t.marginSell = 0.05 + 0.3 * rand.Float64()
	t.margin = 0.0
	//t.marginBuy = 0.0
	//t.marginSell = 0.0
	t.prevBestBidPrice = -1
	t.prevBestAskPrice = -1
	t.job = &TraderOrder{Type: "NA"}
}

func (t *ZIPTrader) setPrice() {
	// round to 2 dp
	t.price = common.Round((t.limitPrice*(1+t.margin))*100.0) / 100.0
}

func (t *ZIPTrader) SetOrders(orders []*TraderOrder) {
	t.Info.ExecutionOrders = orders
	t.limitPrice = orders[0].LimitPrice
	t.active = true
	t.job = orders[0]
	if t.job.IsBid() {
		t.margin = t.marginBuy
	} else {
		t.margin = t.marginSell
	}
	t.limitPrice = t.job.LimitPrice
	quotePrice := t.job.LimitPrice * (1.0 + t.margin)
	t.price = quotePrice
}

func (t *ZIPTrader) AddOrder(order *TraderOrder) {
	t.Info.ExecutionOrders = append(t.Info.ExecutionOrders, order)
}

func (t *ZIPTrader) RemoveOrder() error {
	if len(t.Info.ExecutionOrders) == 0 {
		return errors.New("no order to be removed")
	}

	t.Info.ExecutionOrders = t.Info.ExecutionOrders[:len(t.Info.ExecutionOrders)-1]
	if len(t.Info.ExecutionOrders) != 0 {
		t.limitPrice = t.Info.ExecutionOrders[0].LimitPrice
	}
	return nil
}

func (t *ZIPTrader) GetOrder(timeStep int) *common.Order {
	if len(t.Info.ExecutionOrders) < 1 {
		t.active = false
		t.job = &TraderOrder{Type: "NA"}
		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NA",
		}
	}

	order := t.Info.ExecutionOrders[0]
	if !order.IsValid() {
		err := t.RemoveOrder()
		if err != nil {
			log.WithFields(log.Fields{
				"ExecOrder": order,
				"Place":     "ZIP Trader GetOrder",
			}).Error("Error:", err)
		}

		return &common.Order{
			TraderID:  t.Info.TraderID,
			OrderType: "NAN",
		}
	}

	t.active = true
	t.job = order
	if order.IsBid() {
		t.margin = t.marginBuy
	} else {
		t.margin = t.marginSell
	}

	t.setPrice()
	marketOrder := &common.Order{
		TraderID:  t.Info.TraderID,
		OrderType: order.Type,
		Price:     t.price,
		Quantity:  order.Quantity,
		TimeStep:  timeStep,
		Time:      time.Now(),
	}
	t.Info.ActiveOrders[timeStep] = marketOrder
	return marketOrder
}

func (t *ZIPTrader) MarketUpdate(update common.MarketUpdate) {
	// assumes that a trader can not change from buyer to seller
	// Sellers
	if t.Info.SellerOrBuyer == "SELLER" {
		// If last shout was accepted at price q
		if update.LastTrade.TimeStep == update.TimeStep {
			// Get las accepted shout accepted price and not trade price
			shoutP := update.LastTrade.SellOrder.Price
			if update.LastTrade.BuyOrder.TimeStep == update.TimeStep {
				shoutP = update.LastTrade.BuyOrder.Price
			}

			// seller should raise profit if p < q
			if t.price <= shoutP {
				target := t.targetUp(shoutP)
				t.profitAlter(target)
				// FIXME: Added for debugging only
				t.LogMargin(update.Day, update.TimeStep, update.EID)
			}
			// if last shout was a bid
			if update.LastTrade.BuyOrder.TimeStep == update.TimeStep && t.active &&
				t.price >= shoutP {
				target := t.targetDown(shoutP)
				t.profitAlter(target)
				// FIXME: Added for debugging only
				t.LogMargin(update.Day, update.TimeStep, update.EID)
			}
		} else if update.BestAsk != -1 {
			sort.Sort(common.ByTimeStep(update.Asks))
			lastAsk := update.Asks[len(update.Asks)-1]
			//log.WithFields(log.Fields{
			//	"TID":t.Info.TraderID,
			//	"len asks": len(update.Asks),
			//	"last ask ts": update.Asks[len(update.Asks)-1].TimeStep,
			//	"first ask ts": update.Asks[0].TimeStep,
			//	"Price last ask": update.Asks[len(update.Asks)-1].Price,
			//	"Price": t.price,
			//	"active": t.active,
			//}).Warn("Here5")
			// if (the last shout was an offer)
			// any active seller s_i for which p_i >= q should lower its margin
			if t.active && t.price >= lastAsk.Price {
				target := t.targetDown(lastAsk.Price)
				t.profitAlter(target)
				// FIXME: Added for debugging only
				t.LogMargin(update.Day, update.TimeStep, update.EID)
			}
		}
	} else {
		// Buyers
		// If last shout was accepted at price q
		if update.LastTrade.TimeStep == update.TimeStep {
			// any buyer b_i for which p_i >= q should raise its profit margin
			shoutP := update.LastTrade.SellOrder.Price
			if update.LastTrade.BuyOrder.TimeStep == update.TimeStep {
				shoutP = update.LastTrade.BuyOrder.Price
			}

			if t.price >= shoutP {
				target := t.targetUp(shoutP)
				t.profitAlter(target)
				// FIXME: Added for debugging only
				t.LogMargin(update.Day, update.TimeStep, update.EID)
			}
			// if last shout was a offer  any active buyer b_i for which p_i <= q should lower its margin
			if update.LastTrade.SellOrder.TimeStep == update.TimeStep && t.active &&
				t.price <= shoutP {
				target := t.targetDown(shoutP)
				t.profitAlter(target)
				// FIXME: Added for debugging only
				t.LogMargin(update.Day, update.TimeStep, update.EID)
			}
		} else if update.BestBid != -1 {
			sort.Sort(common.ByTimeStep(update.Bids))
			lastBid := update.Bids[len(update.Bids)-1]
			//log.WithFields(log.Fields{
			//	"TID":t.Info.TraderID,
			//	"len Bids": len(update.Bids),
			//	"lastbid ts": update.Bids[len(update.Bids)-1].TimeStep,
			//	"first bid ts": update.Bids[0].TimeStep,
			//	"Price last bid": update.Bids[len(update.Bids)-1].Price,
			//	"Price": t.price,
			//	"active": t.active,
			//}).Warn("Here11")
			// if (the last shout was a bid)
			// any active buyer s_i for which p_i <= q should lower its margin
			if t.active && t.price <= lastBid.Price {
				target := t.targetDown(lastBid.Price)
				t.profitAlter(target)
				// FIXME: Added for debugging only
				t.LogMargin(update.Day, update.TimeStep, update.EID)
			}
		}
	}
}

func (t *ZIPTrader) willingToTrade(price float64) bool {
	if t.job.IsBid() && t.active && t.price >= price {
		return true
	}

	if t.job.IsAsk() && t.active && t.price <= price {
		return true
	}

	return false
}

func (t *ZIPTrader) targetUp(price float64) float64 {
	//  Generate a higher target price by randomly perturbing given price
	if t.Info.SellerOrBuyer == "SELLER" {
		absolutePerturbation := t.ca * rand.Float64()
		relativePerturbation := price * (1.0 + (t.cr * rand.Float64()))
		target := relativePerturbation + absolutePerturbation
		return target
	} else {
		absolutePerturbation := t.ca * rand.Float64()
		relativePerturbation := price * (1.0 - (t.cr * rand.Float64()))
		target := relativePerturbation - absolutePerturbation
		return target
	}
}

func (t *ZIPTrader) targetDown(price float64) float64 {
	//  Generate a lower target price by randomly perturbing given price
	if t.Info.SellerOrBuyer == "SELLER" {
		absolutePerturbation := t.ca * rand.Float64()
		relativePerturbation := price * (1.0 - (t.cr * rand.Float64()))
		target := relativePerturbation - absolutePerturbation
		return target
	} else {
		absolutePerturbation := t.ca * rand.Float64()
		relativePerturbation := price * (1.0 + (t.cr * rand.Float64()))
		target := relativePerturbation + absolutePerturbation
		return target
	}

}

func (t *ZIPTrader) profitAlter(price float64) {
	oldPrice := t.price
	diff := price - oldPrice
	change := ((1.0 - t.momentum) * (t.beta * diff)) + (t.momentum * t.lastDelta)
	t.lastDelta = change
	newMargin := ((t.price + change) / t.limitPrice) - 1.0

	if t.job.IsBid() {
		if newMargin < 0.0 {
			t.marginBuy = newMargin
			t.margin = newMargin
		}
	} else {
		if newMargin > 0.0 {
			t.marginSell = newMargin
			t.margin = newMargin
		}
	}
	t.setPrice()
}

func (t *ZIPTrader) TradeMade(trade *common.Trade) bool {
	// Got to ZIC.go to read about this function weaknesses and reasoning
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

func (t *ZIPTrader) GetExecutionOrder() []*TraderOrder {
	return t.Info.ExecutionOrders
}

func (t *ZIPTrader) LogBalance(fileName string, day int, trade *common.Trade) {
	fileName, err := filepath.Abs(fileName + "/ZIPTradersLog.csv")
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
		}).Error("ZIP trader CSV file could not be made")
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

func (t *ZIPTrader) LogOrder(fileName string, d, ts, tradeID int, tPrice float64) {
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
		}).Error("ZIP exec order CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if addHeader {
		writer.Write([]string{"Day", "TimeStep", "TID", "TradeID", "LimitPrice", "TPrice", "OType"})
	}

	writer.Write([]string{
		strconv.Itoa(d),
		strconv.Itoa(ts),
		strconv.Itoa(t.Info.TraderID),
		strconv.Itoa(tradeID),
		fmt.Sprintf("%.5f", t.Info.ExecutionOrders[0].LimitPrice),
		fmt.Sprintf("%.5f", tPrice),
		t.Info.ExecutionOrders[0].Type,
	})
}

func (t *ZIPTrader) LogMargin(d, ts int, eid string) {
	// NOTE: stop during GA test to speed up process
	return
	// For now assume agents have only one order at a time
	fileName, err := filepath.Abs("../mexs/logs/" + eid + "/ZIPMargin.csv")
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
		}).Error("ZIP margin CSV file could not be made")
		return
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if addHeader {
		writer.Write([]string{"Day", "TimeStep", "TID", "Type", "LimitPrice", "Price", "Margin"})
	}

	writer.Write([]string{
		strconv.Itoa(d),
		strconv.Itoa(ts),
		strconv.Itoa(t.Info.TraderID),
		t.Info.SellerOrBuyer,
		fmt.Sprintf("%.5f", t.limitPrice),
		fmt.Sprintf("%.5f", t.price),
		fmt.Sprintf("%.5f", t.margin),
	})
}

var _ RobotTrader = (*ZIPTrader)(nil)
