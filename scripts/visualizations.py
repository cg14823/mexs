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

    figSD = plt.figure("Trade Prices")
    ax = figSD.add_subplot(111)
    ax.set_xlabel("Quantity")
    ax.set_ylabel("Price")
    ax.set_xlim((0, (max(quantityD[-1], quantityS[-1])+1)))
    ax.grid()
    ax.xaxis.set_ticks(range(0, max(quantityD[-1], quantityS[-1])+1))
    ax.step(quantityD, demand[:,0], 'r')
    ax.step(quantityS, supply[:,0], 'g')
    

plot_trades("TRADES_ID-43015129-5b78-4150-9521-5797c25289d4_0-300.csv")
plt.show()
plot_supply_demmand("LIMITPRICES_ID-43015129-5b78-4150-9521-5797c25289d4_1.csv")
plt.show()