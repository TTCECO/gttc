## PBFT Consensus Algorithm

Blockchains are inherently decentralized systems which consist of different actors who act depending on their incentives and on the information that is available to them.

Whenever a new transaction gets broadcasted to the network, nodes have the option to include that transaction to their copy of their ledger or to ignore it. When the majority of the actors which comprise the network decide on a single state, consensus is achieved.

```
A fundamental problem in distributed computing and multi-agent systems is to achieve overall system reliability in the presence of a number of faulty processes. This often requires processes to agree on some data value that is needed during computation.
```

These processes are described as consensus.
* What happens when an actor decides to not follow the rules and to tamper with the state of his ledger?
* What happens when these actors are a large part of the network, but not the majority?

Famously described in 1982 by Lamport, Shostak and Pease, it is a generalized version of the Two Generals Problem with a twist. It describes the same scenario, where instead more than two generals need to agree on a time to attack their common enemy. The added complication here is that one or more of the generals can be a traitor, meaning that they can lie about their choice.

The leader-follower paradigm described in the Two Generals Problem is transformed to a commander-lieutenant setup. In order to achieve consensus here, the commander and every lieutenant must agree on the same decision (for simplicity attack or retreat).

Byzantine Generals Problem. A commanding general must send an order to his n-1 lieutenant generals such that

1. All loyal lieutenants obey the same order.
2. If the commanding general is loyal, then every loyal lieutenant obeys the order he sends.


Adding to 2., it gets interesting that if the commander is a traitor, consensus must still be achieved. As a result, all lieutenants take the majority vote.

The algorithm to reach consensus in this case is based on the value of majority of the decisions a lieutenant observes.

```
Theorem: For any m, Algorithm OM(m) reaches consensus if there are more than 3m generals and at most m traitors.
```

This implies that the algorithm can reach consensus as long as 2/3 of the actors are honest. If the traitors are more than 1/3, consensus is not reached, the armies do not coordinate their attack and the enemy wins.

```
The important thing to remember is that the goal is for the majority of the lieutenants to choose the same decision, not a specific one.
```






