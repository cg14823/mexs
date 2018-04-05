import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
import sys
import glob

def plot_elite_score(eid):
    f = '../logs/{}/elite.csv'.format(eid)
    elites = pd.read_csv(filepath_or_buffer=f)

    figE = plt.figure("Elites score")
    ax = figE.add_subplot(111)

    ax.set_ylabel('Score')
    ax.set_xlabel('Gen')

    ax.plot(elites['Gen'], elites['Score'])
    ax.grid()

def plot_gen_scores(eid):
    f = '../logs/{}/chromozones.csv'.format(eid)
    f1 = '../logs/{}/elite.csv'.format(eid)
    elites = pd.read_csv(filepath_or_buffer=f1)
    cs = pd.read_csv(filepath_or_buffer=f)

    figAv = plt.figure('Score evolution')
    ax = figAv.add_subplot(111)

    ax.set_ylabel('Score')
    ax.set_xlabel('Generation')

    ax.plot(elites['Gen'], elites['Score'], color='k', label='Best score')
    ax.grid()

    maxGen = cs['Gen'].max() + 1
    averageScores = []
    stdScores = []
    ubound = []
    lbound = []
    gen = list(range(maxGen))
    for g in range(maxGen):
        gen = cs.loc[(cs['Gen'] == g)]
        averageScores.append(gen['Score'].mean())
        stdScores.append(gen['Score'].std())
        ubound.append(averageScores[-1] + stdScores[-1])
        lbound.append(averageScores[-1] - stdScores[-1])

    ax.plot(elites['Gen'], averageScores, color='r', label='Average Score')
    ax.plot(elites['Gen'], ubound, color='g', label='1 standard deviation')
    ax.plot(elites['Gen'], lbound, color='g')
    ax.legend()

def plot_avg_genes(eid):
    f = '../logs/{}/chromozones.csv'.format(eid)
    cs = pd.read_csv(filepath_or_buffer=f)

    fig = plt.figure('Average Genes')

    maxGen = cs['Gen'].max() + 1
    gen = list(range(maxGen))
    # Bid:ask
    bar = []
    bstd = []
    # K pricing
    kpricing =[]
    kstd =[]
    #Window Size EE
    ws = []
    wstd =[]
    #DeltaEE
    deltaEE = []
    dstd = []
    #MaxShift 
    ms =[]
    mstd = []
    # Dominance
    dominance = []
    domstd = []

    for g in range(maxGen):
        v = cs.loc[cs['Gen'] == g]
        bar.append(v['B:A'].mean())
        bstd.append(v['B:A'].std())

        kpricing.append(v['K'].mean())
        kstd.append(v['K'].std())

        ws.append(v['WindowSizeEE'].mean())
        wstd.append(v['WindowSizeEE'].std())

        deltaEE.append(v['DeltaEE'].mean())
        dstd.append(v['DeltaEE'].std())

        ms.append(v['MaxShift'].mean())
        mstd.append(v['MaxShift'].std())

        dominance.append(v['Dominance'].mean())
        domstd.append(v['Dominance'].std())

    axba = fig.add_subplot(231)
    axba.plot(gen, bar, color='b', label='bid ask ratio')
    axba.plot(gen, [a-b for a,b in zip(bar, bstd)], color='g', label='1 std')
    axba.plot(gen, [a+b for a,b in zip(bar, bstd)], color='g')
    axba.legend()
    axba.grid()


    axk = fig.add_subplot(232)
    axk.plot(gen, kpricing, color='b', label='K-Pricing')
    axk.plot(gen, [a-b for a,b in zip(kpricing, kstd)], color='g', label='1 std')
    axk.plot(gen, [a+b for a,b in zip(kpricing, kstd)], color='g')
    axk.legend()
    axk.grid()


    axw = fig.add_subplot(233)
    axw.plot(gen, ws, color='b', label='Window Size')
    axw.plot(gen, [a-b for a,b in zip(ws, wstd)], color='g', label='1 std')
    axw.plot(gen, [a+b for a,b in zip(ws, wstd)], color='g')
    axw.legend()
    axw.grid()

    axde = fig.add_subplot(234)
    axde.plot(gen, deltaEE, color='b', label='Delta')
    axde.plot(gen, [a-b for a,b in zip(deltaEE, dstd)], color='g', label='1 std')
    axde.plot(gen, [a+b for a,b in zip(deltaEE, dstd)], color='g')
    axde.legend()
    axde.grid()

    axm = fig.add_subplot(235)
    axm.plot(gen, ms, color='b', label='Max Shift')
    axm.plot(gen, [a-b for a,b in zip(ms, mstd)], color='g', label='1 std')
    axm.plot(gen, [a+b for a,b in zip(ms, mstd)], color='g')
    axm.legend()
    axm.grid()

    axdo = fig.add_subplot(236)
    axdo.plot(gen,dominance, color='b', label='Dominance')
    axdo.plot(gen, [a-b for a,b in zip(dominance, domstd)], color='g', label='1 std')
    axdo.plot(gen, [a+b for a,b in zip(dominance, domstd)], color='g')
    axdo.legend()
    axdo.grid()

def plot_elite_genes(eid):
    pass

def trades(eid, ep=None):
    f = '../logs/{}/TRADES.csv'.format(eid)

    trades = pd.read_csv(filepath_or_buffer=f)
    maxDay = trades['TradingDay'].max()
    for d in range(maxDay + 1):
        ts = trades.loc[(trades['TradingDay'] == d)]

        figT = plt.figure("Trade Prices day {} / {}:".format(d, maxDay))
        ax = figT.add_subplot(111)

        meanPrice = np.mean(ts["Price"])
        maxTimeStep = np.max(ts["TimeStep"])
        if ts["TimeStep"].count() == 0:
            maxTimeStep = 10
        timeSteps = [0, maxTimeStep + 1]

        ax.plot(ts["TimeStep"], ts["Price"], 'ro', label="Trades")
        ax.plot(ts["TimeStep"], ts["Price"], 'r')
        ax.plot(timeSteps, [meanPrice, meanPrice], label="Mean trade price")
        if (not ep == None) and (not ep == []):
            ax.plot(timeSteps, [ep[0], ep[0]], linestyle ='--', color='g', label="Equilibirum Price")
        ax.set_xlabel("Time Steps")
        ax.set_ylabel("Price")
        ax.set_xlim(timeSteps[0], timeSteps[1])
        ax.grid()
        ax.legend()

def supplyDemand(eid):
    LpsF = '../logs/'+eid+'/LimitPrices.csv'
    SchedsF = '../logs/'+eid+'/schedule.csv'

    lps = pd.read_csv(filepath_or_buffer=LpsF)
    sched = pd.read_csv(filepath_or_buffer=SchedsF)
    schedIds = sched['ScheduleID'].unique()

    prices_times = []
    for id in schedIds:
        demand = lps.loc[(lps["TYPE"] == "BID") & (lps["ID"] == id)].sort_values(by=['LIMIT'],ascending=False).as_matrix(['LIMIT'])
        supply = lps.loc[(lps["TYPE"] == "ASK") & (lps["ID"] == id)].sort_values(by=['LIMIT'],ascending=True).as_matrix(['LIMIT'])

        quantityD = np.asarray(range(0,len(demand[:,0])+1, 1))
        quantityS = np.asarray(range(0,len(supply[:,0])+1, 1))
        found = False
        si =0
        di =0
        pe = None
        qe = None
        while not found :
            if (si >= len(supply[:,0])) or (di >= len(demand[:,0])):
                break
            if demand[di] > supply[si]:
                si += 1
                di +=1
                continue
            elif demand[di] == supply[si]:
                found =True
                pe = demand[di]
                qe = si 
            else:
                found = True
                pe = (demand[di] +supply[si]) / 2.0
                qe = (di + si)/2.0

       
        figSD = plt.figure("Supply demand curve #{}".format(id))
        ax = figSD.add_subplot(111)
        ax.set_xlabel("Quantity")
        ax.set_ylabel("Price")
        ax.set_xlim((0, (max(quantityD[-1], quantityS[-1])+1)))
        ax.set_ylim(min(demand[-1], supply[0]) - 5, max(demand[0], supply[-1]) + 5)
        ax.grid()
        dee = demand[:,0].tolist()
        suu = supply[:,0].tolist()
        dee = [dee[0]] + dee
        suu = [suu[0]] + suu
        print(dee)
        step = 1
        limit = max(quantityD[-1], quantityS[-1])+1
        if limit >= 50:
            step = 2
        if limit >=100:
            step = 5
        ax.xaxis.set_ticks(range(0, limit, step))
        ax.step(quantityD, dee, 'r', label="demand")
        ax.step(quantityS, suu, 'g', label="supply")

        if found:
            ax.plot([0, qe], [pe, pe], linestyle='--', color="b")
            ax.plot([qe,qe], [0, pe], linestyle='--', color="b")
            print(pe)
            prices_times.append(pe)
        ax.legend()
    
    return prices_times


def main():
    print(sys.argv)
    if len(sys.argv) == 3:
        print("Here")
        action = sys.argv[1]
        eid = sys.argv[2]
        
        if action == "trades":
            print("heree")
            trades(eid)
        elif action == "sd":
            supplyDemand(eid)
        elif action == "sdt":
            pricesTimes = supplyDemand(eid)
            trades(eid, pricesTimes)
        elif action == "gen-score":
            plot_gen_scores(eid)
            plot_avg_genes(eid)
        else:
            print("Commands [sd, sdt, gen-score]")
            return
        plt.show()

if __name__ == "__main__":
    main()

