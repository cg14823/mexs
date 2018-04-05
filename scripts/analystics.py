import numpy as np
import pandas as pd
import json
import sys


def ibmTest(rootP, root1, its=100, days=5):
    efficencys = np.zeros((days, its))
    numtrades = np.zeros((days, its))
    avgTps = np.zeros((days, its))
    alphas = np.zeros((days, its))

    # FIXME: hardocded max surplus and equilibirum price and max num of trades

    DMaxSurplus = 62.5
    SMaxSurplus = 62.5
    maxSurplus = DMaxSurplus + SMaxSurplus

    ep = 122.5
    tmax = 5

    ep = 157.5
    tmax = 6
    for i in range(its):
        fTrades = '{}{}/TRADES.csv'.format(rootP, i)
        trades = pd.read_csv(filepath_or_buffer=fTrades)

        # FIXME: Hardcoded number of days
        for d in range(days):
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
                summ += (x['Price'] -x['SL']) + (x['BL'] - x['Price'])
                
            efficencys[d, i] = (summ / maxSurplus)
            
    # FIXME: should created via a for loop and not hardcode 5 entries as not always 5 days
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

def effSingle(rootP, days=5):
    fTrades = '{}/TRADES.csv'.format(rootP)
    trades = pd.read_csv(filepath_or_buffer=fTrades)

    efficencys = np.zeros((days,))
    numtrades = np.zeros((days,))
    avgTps = np.zeros((days,))
    alphas = np.zeros((days,))

    DMaxSurplus = 62.5
    SMaxSurplus = 62.5
    maxSurplus = DMaxSurplus + SMaxSurplus

    ep = 148
    tmax = 46

    # FIXME: Hardocoded number of days
    for d in range(days):
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
           summ += (x['Price'] -x['SL']) + (x['BL'] - x['Price'])

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

def calculate_max_surplus(sellers, buyers, pe):
    sellerProfit = 0.0
    buyersProfit = 0.0
    
    # sort sellers limit prices in ascending order and buyers in descending
    sellers.sort()
    buyers.sort(reverse=True)

    for s in sellers:
        if s <= pe:
            sellerProfit += pe - s
        else:
            break

    for b in buyers:
        if b >= pe:
            buyersProfit += b - pe
        else:
            break

    return sellerProfit, buyersProfit

def calculate_equilibrium(sellers, buyers):
    pe = None
    eq = None

    # sort sellers limit prices in ascending order and buyers in descending
    sellers.sort()
    buyers.sort(reverse=True)

    found = False
    ix = 0 
    while not found:
        if (ix >= len(sellers) or ix >= len(buyers)):
            break
        elif sellers[ix] == buyers[ix]:
            pe = sellers[ix]
            eq = ix
            break
        elif buyers[ix] < sellers[ix]:
            pe = (buyers[ix] + sellers[ix]) / 2.0
            eq = ix
            break
        ix += 1

    return pe, eq


def eq_command(eid, verbose=True):
    LpsF = '../logs/'+eid+'/LimitPrices.csv'
    SchedsF = '../logs/'+eid+'/schedule.csv'

    lps = pd.read_csv(filepath_or_buffer=LpsF)
    sched = pd.read_csv(filepath_or_buffer=SchedsF)
    schedIds = sched['ScheduleID'].unique()

    results = dict()

    for sid in schedIds:
        if sid in results.keys():
            continue

        sellers = lps.loc[(lps['TYPE'] == 'ASK') & (lps['ID'] == sid)].as_matrix(['LIMIT']).flatten().tolist()
        buyers = lps.loc[(lps['TYPE'] == 'BID') & (lps['ID'] == sid)].as_matrix(['LIMIT']).flatten().tolist()
     
        eqP, eqQ = calculate_equilibrium(sellers, buyers)
        sMaxProfit, bMaxProfit = calculate_max_surplus(sellers, buyers, eqP)
        results[str(sid)] = {'eqP': eqP, 'eqQ':eqQ, 'sMaxProfit': sMaxProfit, 'bMaxProfit': bMaxProfit}

    if verbose:
        print(results)
    return results


def all_command(eid, days=5):
    results = eq_command(eid)
    f = '../logs/{}/TRADES.csv'.format(eid)
    trades = pd.read_csv(filepath_or_buffer=f)
    schedf = '../logs/{}/schedule.csv'.format(eid)
    sched = pd.read_csv(filepath_or_buffer=schedf)

    efficencys = np.zeros((days,))
    numtrades = np.zeros((days,))
    avgTps = np.zeros((days,))
    alphas = np.zeros((days,))

    for d in range(days):
        # For now only look at first schedule
        ids = sched.loc[sched['TradingDay'] == d].as_matrix(['ScheduleID']).flatten().tolist()
        metadata = results[str(ids[0])]

        tradesDi = trades.loc[(trades['TradingDay'] == d)]
        # get number of trades in day d
        numtrades[d] = len(tradesDi.index) / metadata['eqQ']
        # get avg transaction price
        avgTps[d] = tradesDi['Price'].mean()

        summ = 0
        for p in tradesDi['Price']:
            summ += (p - metadata['eqP']) ** 2

        if len(tradesDi.index) > 0:
            summ / len(tradesDi.index)

        # caluclate alpha for day d in experiment i
        alphas[d] = (100.0 / metadata['eqP']) * (summ **0.5)

        # get efficency for day d in experiment i
        summ = 0.0
        for ix, x in tradesDi.iterrows():
           summ += (x['Price'] -x['SL']) + (x['BL'] - x['Price'])
        
        efficencys[d] = (summ / (metadata['sMaxProfit'] + metadata['bMaxProfit']))
    
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
        'data': results,
    }

    with open('../logs/'+eid+'/analytics.json', 'w') as out:
        json.dump(data, out, indent=4)

def multiRun_command(eid, days=5, its=100):
    efficencys = np.zeros((days, its))
    numtrades = np.zeros((days, its))
    avgTps = np.zeros((days, its))
    alphas = np.zeros((days, its))
    
    for i in range(its):
        runEid = eid+"/runs_"+str(i)
        results = eq_command(runEid, False)
        f = '../logs/{}/TRADES.csv'.format(runEid)
        trades = pd.read_csv(filepath_or_buffer=f)
        schedf = '../logs/{}/schedule.csv'.format(runEid)
        sched = pd.read_csv(filepath_or_buffer=schedf)

        for d in range(days):
            # NOTE For now only look at first schedule
            ids = sched.loc[sched['TradingDay'] == d].as_matrix(['ScheduleID']).flatten().tolist()
            metadata = results[str(ids[0])]

            tradesDi = trades.loc[(trades['TradingDay'] == d)]
            # get number of trades in day d
            numtrades[d, i] = len(tradesDi.index) / metadata['eqQ']
            # get avg transaction price
            avgTps[d, i] = tradesDi['Price'].mean()

            summ = 0
            for p in tradesDi['Price']:
                summ += (p - metadata['eqP']) ** 2

            if len(tradesDi.index) > 0:
                summ / len(tradesDi.index)

            # caluclate alpha for day d in experiment i
            alphas[d, i] = (100.0 / metadata['eqP']) * (summ **0.5)

            # get efficency for day d in experiment i
            summ = 0.0
            for ix, x in tradesDi.iterrows():
                summ += (x['Price'] -x['SL']) + (x['BL'] - x['Price'])
            
            efficencys[d, i] = (summ / (metadata['sMaxProfit'] + metadata['bMaxProfit']))

    meanEffs = []
    meanNumTrades = []
    meanalphas = []
    meanTps = []
    stdsTps= []

    for d in range(days):
        meanEffs.append(np.nanmean(efficencys[d,:]))
        meanNumTrades.append(np.nanmean(numtrades[d,:]))
        meanalphas.append(np.nanmean(alphas[d,:]))
        meanTps.append(np.nanmean(avgTps[d,:]))
        stdsTps.append(np.nanstd(avgTps[d,:]))

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
        'data': results
    }

    with open('../logs/'+eid+'/analytics.json', 'w') as out:
        json.dump(data, out, indent=4)




def main():
    """
    Possible actions are:
        - eq : gets equilibrium price, equilibrium quantity and max surplus
        - all: gets all statts for single stat
        - multiRun gets all stats for multirun
        - GA: gets all stats for GA
    """

    if len (sys.argv) < 3:
        print("To run please use p3.6 anayltics [COMMAND] [OPTIONS]")
        print("COMANDS :- eq, all, multi-run, ga")
        print("options :- [eid, days, runs]")
    else:
        action = sys.argv[1]
        eid = sys.argv[2]
        if action == "eq":
            eq_command(eid)
        elif action == "all":
            if len(sys.argv) >= 4:
                days = int(sys.argv[3])
                all_command(eid, days)
            else:
                all_command(eid)
        elif action=="multi-run":
            days=5
            its =100
            if len(sys.argv) >= 4:
                days = int(sys.argv[3])
                if len(sys.argv >=5):
                    its=int(sys.argv[4])
            multiRun_command(eid, days=days, its=its)




if __name__ == "__main__":
    main()
