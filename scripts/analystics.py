import numpy as np
import pandas as pd
import json

def ibmTest(rootP, root1):
    efficencys = np.zeros((5, 100))
    numtrades = np.zeros((5, 100))
    avgTps = np.zeros((5, 100))
    alphas = np.zeros((5, 100))

    # FIXME: hardocded max surplus and equilibirum price and max num of trades
    DMaxSurplus = 114
    SMaxSurplus = 221
    maxSurplus = DMaxSurplus + SMaxSurplus

    ep = 157.5
    tmax = 6

    # FIXME: Hardcoded number of test

    for i in range(100):
        fTrades = '{}{}/TRADES.csv'.format(rootP, i)
        fOrders = '{}{}/ExecOrders.csv'.format(rootP, i) 
        trades = pd.read_csv(filepath_or_buffer=fTrades)
        orders = pd.read_csv(filepath_or_buffer=fOrders)

        # FIXME: Hardcoded number of days
        for d in range(5):
            tradesDi = trades.loc[(trades['TradingDay'] == d)]
            # get number of trades in day d
            numtrades[d, i] = len(tradesDi.index) / tmax
            # get avg transaction price
            avgTps[d, i] = tradesDi['Price'].mean()
            
            summ = 0
            for p in tradesDi['Price']:
               summ += (p - ep) ** 2

            if len(tradesDi.index) > 0:
                summ / len(tradesDi.index)

            # caluclate alpha for day d in experiment i
            alphas[d,i] = (100.0 / ep) * (summ **0.5)

            # get efficency for day d in experiment i
            summ = 0.0
            for ix, x in tradesDi.iterrows():
                execOrderB = orders.loc[(orders['TradeID'] == x['ID']) & (orders['Day'] == d) & (orders['OType'] == 'BID')]
                execOrderA = orders.loc[(orders['TradeID'] == x['ID']) & (orders['Day'] == d) & (orders['OType'] == 'ASK')]
                summ += (x['Price'] -execOrderA.iloc[0]['LimitPrice']) +   (execOrderB.iloc[0]['LimitPrice'] - x['Price'])
                
            efficencys[d, i] = (summ / maxSurplus)

    meanEffs = [np.nanmean(efficencys[0,:]), np.nanmean(efficencys[1,:]), np.nanmean(efficencys[2,:]), np.nanmean(efficencys[3,:]), np.nanmean(efficencys[4,:]) ]
    meanNumTrades = [numtrades[0,:].mean(), numtrades[1,:].mean(), numtrades[2,:].mean(), numtrades[3,:].mean(), numtrades[4,:].mean() ]
    meanalphas = [np.nanmean(alphas[0,:]), np.nanmean(alphas[1,:]), np.nanmean(alphas[2,:]), np.nanmean(alphas[3,:]), np.nanmean(alphas[4,:]) ]
    meanTps = [np.nanmean(avgTps[0,:]), np.nanmean(avgTps[1,:]), np.nanmean(avgTps[2,:]), np.nanmean(avgTps[3,:]),np.nanmean(avgTps[4,:]) ]
    stdsTps = [np.nanstd(avgTps[0,:]), np.nanstd(avgTps[1,:]), np.nanstd(avgTps[2,:]), np.nanstd(avgTps[3,:]), np.nanstd(avgTps[4,:]) ]

    print("Effs:", meanEffs)
    print("Trade ratios:",meanNumTrades)
    print("Alphas:",meanalphas)
    print("AveragePrices:",meanTps)
    print("AveragePrices std:",stdsTps)

    efficencys = efficencys.flatten()
    numtrades = numtrades.flatten()
    alphas = alphas.flatten()
    avgTps = avgTps.flatten()

    print('--------Final Results-------')
    print('Efficency: {:.3f} +- {:.3f}'.format(np.nanmean(efficencys), np.nanstd(efficencys)))
    print('Trade Ratio: {:.3f} +- {:.3f}'.format(numtrades.mean(), numtrades.std()))
    print('Avg Price: {:.3f} +- {:.3f}'.format(np.nanmean(avgTps), np.nanstd(avgTps)))
    print('Alpha: {:.3f} +- {:.3f}'.format(np.nanmean(alphas), np.nanstd(alphas)))

    data = {
        'eff': np.nanmean(efficencys),
        'effStd': np.nanstd(efficencys),
        'tr': numtrades.mean(),
        'trStd': numtrades.std(),
        'avgP': np.nanmean(avgTps),
        'avgPStd': np.nanstd(avgTps),
        'alpha': np.nanmean(alphas),
        'alphaStd': np.nanstd(alphas),
        'effsPD': meanEffs,
        'trPD': meanNumTrades,
        'avgPPD': meanTps, 
        'alphasPD': meanalphas,
        'EP': ep,
        'max_surplus': maxSurplus
    }

    with open(root1+'analytics.json', 'w') as out:
        json.dump(data, out, indent=4)

def effSingle(rootP):
    fTrades = '{}/TRADES.csv'.format(rootP)
    fOrders = '{}/ExecOrders.csv'.format(rootP) 
    trades = pd.read_csv(filepath_or_buffer=fTrades)
    orders = pd.read_csv(filepath_or_buffer=fOrders)

    efficencys = np.zeros((5,))
    numtrades = np.zeros((5,))
    avgTps = np.zeros((5,))
    alphas = np.zeros((5,))

    DMaxSurplus = 62.5
    SMaxSurplus = 62.5
    maxSurplus = DMaxSurplus + SMaxSurplus

    ep = 122.5
    tmax = 5

    # FIXME: Hardocoded number of days
    for d in range(5):
        tradesDi = trades.loc[(trades['TradingDay'] == d)]
        # get number of trades in day d
        numtrades[d] = len(tradesDi.index) / tmax
        # get avg transaction price
        avgTps[d] = tradesDi['Price'].mean()
        
        summ = 0
        for p in tradesDi['Price']:
            summ += (p - ep) ** 2

        if len(tradesDi.index) > 0:
            summ / len(tradesDi.index)

        # caluclate alpha for day d in experiment i
        alphas[d] = (100.0 / ep) * (summ **0.5)

        # get efficency for day d in experiment i
        summ = 0.0
        for ix, x in tradesDi.iterrows():
            execOrderB = orders.loc[(orders['TradeID'] == x['ID']) & (orders['Day'] == d) & (orders['OType'] == 'BID')]
            execOrderA = orders.loc[(orders['TradeID'] == x['ID']) & (orders['Day'] == d) & (orders['OType'] == 'ASK')]
            summ += (x['Price'] -execOrderA.iloc[0]['LimitPrice']) +   (execOrderB.iloc[0]['LimitPrice'] - x['Price'])
            
        efficencys[d] = (summ / maxSurplus)


    efficencys = efficencys.flatten()
    numtrades = numtrades.flatten()
    alphas = alphas.flatten()
    avgTps = avgTps.flatten()

    print("Effs:", efficencys)
    print("Trade ratios:",numtrades)
    print("AveragePrices:", avgTps)
    print("Alphas:", alphas)

    print('--------Final Results-------')
    print('Efficency: {:.3f} +- {:.2f}'.format(np.nanmean(efficencys), np.nanstd(efficencys)))
    print('Trade Ratio: {:.3f} +- {:.2f}'.format(numtrades.mean(), numtrades.std()))
    print('Avg Price: {:.3f} +- {:.2f}'.format(np.nanmean(avgTps), np.nanstd(avgTps)))
    print('Alpha: {:.3f} +- {:.2f}'.format(np.nanmean(alphas), np.nanstd(alphas)))

    data = {
        'eff': np.nanmean(efficencys),
        'effStd': np.nanstd(efficencys),
        'tr': numtrades.mean(),
        'trStd': numtrades.std(),
        'avgP': np.nanmean(avgTps),
        'avgPStd': np.nanstd(avgTps),
        'alpha': np.nanmean(alphas),
        'alphaStd': np.nanstd(alphas),
        'effsPD': efficencys.tolist(),
        'trPD': numtrades.tolist(),
        'avgPPD': avgTps.tolist(), 
        'alphasPD': alphas.tolist(),
        'EP': ep,
        'max_surplus': maxSurplus
    }

    with open(rootP+'/analytics.json', 'w') as out:
        json.dump(data, out, indent=4)

ibmTest('../logs/IBMTest/runs__', '../logs/IBMTest/')
#ibmTest('../logs/ZIPDebug100/runs_', '../logs/ZIPDebug100/')
#effSingle('../logs/mnominincrement')