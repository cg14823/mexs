import numpy as np

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

def get_iprices_and_eq(starts=100, ends=200, startb=100, endb=200, ns=10, nb=10):
    sellers = uniform_random_iprices(starts, ends, ns)
    buyers = uniform_random_iprices(startb, endb, nb)
    eq = calulcate_eq_price(sellers, buyers)
    return sellers, buyers, eq


sellers, buyers, eq = get_iprices_and_eq()

print("sellers: ", sellers)
print("buyers: ", buyers)
print("EQ:", eq)