import matplotlib.pyplot as plt
import numpy as np

def plot_trades(filename):
    trades = np.genfromtxt(fname="../logs/"+filename, delimiter=",", names=True)
    fig = plt.figure("Trade Prices")
    ax = fig.add_subplot(111)
    ax.plot(trades["TimeStep"], trades["Price"], 'ro')
    ax.plot(trades["TimeStep"], trades["Price"], 'r')
    ax.set_xlabel("Time Steps")
    ax.set_ylabel("Price")
    plt.show()


plot_trades("TRADES_ID-f2508b71-7130-4b5f-b186-5cedb22a7151_0-300.csv")