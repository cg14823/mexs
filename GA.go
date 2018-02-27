package main

import (
	"mexs/exchange"
	"math/rand"
	log "github.com/sirupsen/logrus"
	"fmt"
	"path/filepath"
	"os"
	"encoding/csv"
	"strconv"
)

type GA struct {
	// Number of individuals in each gen
	N int
	Gens int
	Config ExperimentConfig
	CurrentGen int
	currentGenes []exchange.AuctionParameters
}

func (g *GA) Start() {
	log.Info("HEREEE!")
	// This function will be the heart of the GA
	// Number of individuals in each generation
	err := os.MkdirAll("../mexs/logs/"+g.Config.EID+"/", 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err.Error(),
		}).Error("Log Folder for this experiment could not be made")
	}
	// Things that remain constant per generation
	log.Infof("HEREEE!1%#v", g)
	// START BY initializing the chromosomes
	cs := make([]exchange.AuctionParameters, g.N)
	for i:=0; i< g.N; i++ {

		cs[i] = InitializeChromozones("RANDOM")
	}
	g.currentGenes = cs

	for i:=0; i < g.Gens; i++{
		log.Info("GEN:",i)
		g.chromozonesToCSV(i, g.currentGenes, []float64{1.0,2.0})
	}


}

func (g *GA) FitnessFunction(fnName string) float64 {
	// Allow for different functions to be used
	switch fnName{
	case "ALPHA":
		return 0.0
	case "ALOC-EFF":
		return 0.0
	case "AVG-TRADER-EFF":
		return 0.0
	case "COM-EFFICENCY":
		return 0.0
	default:
		// default to whatever you prefer
		return 0.0
	}
}

func (g *GA) chromozonesToCSV (gen int, cs []exchange.AuctionParameters, scores []float64){
	fileName, err := filepath.Abs(fmt.Sprintf("../mexs/logs/%s/chromozones", g.Config.EID))
	if err != nil {
		log.WithFields(log.Fields{
			"experimentID": g.Config.EID,
			"error":        err.Error(),
		}).Error("File Path not found")
		return
	}
	addheader := true
	if _, err := os.Stat(fileName); err == nil {
		addheader = false
	}

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if addheader {
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


func InitializeChromozones(initType string) exchange.AuctionParameters{
	switch initType {
	case "LOW":
		return exchange.AuctionParameters{
			BidAskRatio: 0.1,
			KPricing: 0,
			MinIncrement: 0,
			MaxShift: 0.1,
			Dominance: 0,
			OrderQueuing: 1,
		}
	case "NORMAL":
		return exchange.AuctionParameters{
			BidAskRatio: 1,
			KPricing: 0.5,
			MinIncrement: 1,
			MaxShift: 2,
			Dominance: 0,
			OrderQueuing: 1,
		}
	case "HIGH":
		return exchange.AuctionParameters{
			BidAskRatio: 10,
			KPricing: 1,
			MinIncrement: 10,
			MaxShift: 100,
			Dominance: 10,
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
			BidAskRatio: rand.Float64() + 0.5,
			KPricing: rand.Float64(),
			MinIncrement: float64(rand.Intn(5)),
			MaxShift: rand.Float64() + 0.5,
			Dominance: rand.Intn(5),
			OrderQueuing: 1,
		}
	default:
		// Note: default to any especial case I want to test out
		return exchange.AuctionParameters{
			BidAskRatio: rand.Float64() + 0.5,
			KPricing: rand.Float64(),
			MinIncrement: float64(rand.Intn(5)),
			MaxShift: rand.Float64() + 0.5,
			Dominance: rand.Intn(5),
			OrderQueuing: 1,
		}
	}
}