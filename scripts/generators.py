import numpy as np
import matplotlib.pyplot as plt


def divide_in_xs(slp, blp, xs =10):
    sellers = [sorted(slp[x:x+xs]) for x in range(0, len(slp), xs)]
    buyers = [sorted(blp[x:x+xs], reverse=True) for x in range(0, len(blp), xs)]
    return sellers, buyers

def uniform_random_fprices(start=100, end=200, n=10):
    return np.random.uniform(start, end, n)

def uniform_random_iprices(start=100, end=200, n=10):
    return np.random.random_integers(start, end, n)

def calulcate_eq_price(sPrices=[], bPrices=[]):
    bi = 0
    eqP = None
    sSorted = sorted(sPrices)
    bSorted = sorted(bPrices, reverse=True)

    for sp in sSorted:
        if bi == len(bSorted):
            return None
        if sp > bSorted[bi]:
            return float(sp + bSorted[bi])/2.0
        elif sp == bSorted[bi]:
            return sp
        else:
            bi += 1
    
    return eqP

def get_iprices_and_eq(starts=100, ends=200, startb=100, endb=200, ns=100, nb=100):
    sellers = uniform_random_iprices(starts, ends, ns)
    buyers = uniform_random_iprices(startb, endb, nb)

    eq = calulcate_eq_price(sellers, buyers)
   

    sSorted = sorted(sellers)
    bSorted = sorted(buyers, reverse=True)

    figSD = plt.figure("Supply and Demand")
    ax = figSD.add_subplot(111)
    ax.set_xlabel("Quantity")
    ax.set_ylabel("Price")
    ax.set_ylim(min(bSorted[-1], sSorted[0]) - 5, max(bSorted[0], sSorted[-1]) + 5)

    dee = [bSorted[0]] + bSorted
    suu = [sSorted[0]] + sSorted

    step = 1
    limit = max(ns, nb)+1
    if limit >= 50:
        step = 2
    if limit >=100:
        step = 5
    ax.xaxis.set_ticks(range(0, limit, step))
    ax.step(list(range(0,nb+1, 1)), dee, 'r', label="demand")
    ax.step(list(range(0,ns+1, 1)), suu, 'g', label="supply")

    ax.plot([0, nb], [eq, eq], linestyle='--', color="b")
    


    ax.set_xlim((0, (max(ns, nb)+1)))
    ax.grid()
    plt.show()

    sellers, buyers = divide_in_xs(sellers, buyers)
    return sellers, buyers, eq


sellers, buyers, eq = get_iprices_and_eq()

for x in sellers:
    print("s :", x)

for y in buyers:
    print("b:", y)

print("EQ:", eq)