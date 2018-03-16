import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

def visualize_profit_margin(eid):
    f = '../logs/{}/ZIPMargin.csv'.format(eid)
    margins = pd.read_csv(filepath_or_buffer=f)

    marginBuyers = margins.loc[margins['Type'] == 'BUYER']
    marginSellers = margins.loc[margins['Type'] == 'SELLER']

    fig = plt.figure("margins:")
    ax = fig.add_subplot(121)
    ax.set_xlim(0, margins['TimeStep'].max() +1)

    for tid in range(10):
        tm = marginSellers.loc[marginSellers['TID'] == tid]
        ax.plot(tm['TimeStep'], tm['Margin'])
    ax2 = fig.add_subplot(122)
    for tid in range(10,20):
        tm = marginBuyers.loc[marginBuyers['TID'] == tid]
        ax2.plot(tm['TimeStep'], tm['Margin'])

    plt.show()

visualize_profit_margin('IBMTest/runs__0')
