import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

def plot_trades(filename):
    trades = np.genfromtxt(fname="../logs/"+filename, delimiter=",", names=True)
    figT = plt.figure("Trade Prices")
    ax = figT.add_subplot(111)
    ax.plot(trades["TimeStep"], trades["Price"], 'ro')
    ax.plot(trades["TimeStep"], trades["Price"], 'r')
    ax.set_xlabel("Time Steps")
    ax.set_ylabel("Price")
    


def plot_supply_demmand(filename):
    trades = pd.read_csv(filepath_or_buffer="../logs/"+filename)
    demand = trades.loc[(trades["TYPE"] == "BID") & (trades["NUMBER"] == 1)].sort_values(by=['LIMIT_PRICE'],ascending=False).as_matrix(['LIMIT_PRICE'])
    supply = trades.loc[(trades["TYPE"] == "ASK") & (trades["NUMBER"] == 1)].sort_values(by=['LIMIT_PRICE'],ascending=True).as_matrix(['LIMIT_PRICE'])

    quantityD = np.asarray(range(0,len(demand[:,0]), 1))
    quantityS = np.asarray(range(0,len(supply[:,0]), 1))

    figSD = plt.figure("Supply demand curve")
    ax = figSD.add_subplot(111)
    ax.set_xlabel("Quantity")
    ax.set_ylabel("Price")
    ax.set_xlim((0, (max(quantityD[-1], quantityS[-1])+1)))
    ax.grid()
    ax.xaxis.set_ticks(range(0, max(quantityD[-1], quantityS[-1])+1))
    ax.step(quantityD, demand[:,0], 'r')
    ax.step(quantityS, supply[:,0], 'g')
    

plot_trades("TRADES_ID-4cf6b4a4-4a89-4274-8f31-9747aca91dfb_0-50.csv")
plot_supply_demmand("LIMITPRICES_ID-4cf6b4a4-4a89-4274-8f31-9747aca91dfb_1.csv")
plt.show()