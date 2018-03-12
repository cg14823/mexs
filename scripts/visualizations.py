import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
import sys
import glob

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
    files = glob.glob('../logs/'+eid+'/LIMITPRICES_*.csv')
    prices_times = []
    for f in files:
        trades = pd.read_csv(filepath_or_buffer=f)
        demand = trades.loc[(trades["TYPE"] == "BID") & (trades["NUMBER"] == 0)].sort_values(by=['LIMIT_PRICE'],ascending=False).as_matrix(['LIMIT_PRICE'])
        supply = trades.loc[(trades["TYPE"] == "ASK") & (trades["NUMBER"] == 0)].sort_values(by=['LIMIT_PRICE'],ascending=True).as_matrix(['LIMIT_PRICE'])

        quantityD = np.asarray(range(0,len(demand[:,0])+1, 1))
        quantityS = np.asarray(range(0,len(supply[:,0])+1, 1))

        # #  This only work for linearish graphs, if the graphs where exponential another formula should be used
        # # Better way would be using a line intersection algorithm
        # # Pe = gradentSuppy * Qe + Cs = gradentDemant * Qe +Cd
        # # Qe = (Cd -Cs) / (gradeintSupply -gradientDemand)
        # gradientSupply = float(supply[-1] -supply[0])/ float(len(supply[:,0]) - 1)
        # gradientDemand = float(demand[-1] -demand[0])/ float(len(demand[:,0]) - 1)
        # Cs = supply[0]
        # Cd = demand[0]
        # print(Cs)
        # print(Cd)
        # Qe =  float(Cd -Cs) / float(gradientSupply -gradientDemand)
        # Pe = gradientSupply *Qe +Cs
        # print(Qe)
        # print(Pe)
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

        number = f[f.find('_')+1:f.find('.csv')]
        figSD = plt.figure("Supply demand curve #{}".format(number))
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
        ax.xaxis.set_ticks(range(0, max(quantityD[-1], quantityS[-1])+1))
        ax.step(quantityD, dee, 'r', label="demand")
        ax.step(quantityS, suu, 'g', label="supply")

        if found:
            ax.plot([0, qe], [pe, pe], linestyle='-', color="b")
            ax.plot([qe,qe], [0, pe], linestyle='-', color="b")
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
        else:
            pricesTimes = supplyDemand(eid)
            trades(eid, pricesTimes)
        
        plt.show()

if __name__ == "__main__":
    main()

