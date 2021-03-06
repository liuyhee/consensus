# Filecoin Consensus
---

One of Filecoin's main goals is to create a useful Proof-of-Work based on storage, building upon past work in both Proof-of-Work and Proof-of-Stake protocols.

**Disclaimer:** While we work hard to document our work as it progresses, research progress may not be fully reflected here for some time, or may be worked out out-of-band.

This repository houses a lot of our work on this area of research: Filecoin consensus. While it is by no means exhaustive, it should provide a good place from which to start engaging on Filecoin consensus research. You may also want to read up on [Filecoin](https://github.com/filecoin-project/specs) and [Filecoin Research](https://github.com/filecoin-project/research).

Broadly, our goals are to:
- Finalize design aspects of consensus starting with EC to make it secure and workable for wanted Filecoin design,
- Formalize parameters and other implementation requirements in a clear Filecoin Ccnsensus spec implementable by a dev team,
- Define and prove Filecoin consensus security properties.

**Note 1**: Content here may be out-of-sync.

**Note 2**: We may sometimes link to inaccessible content here, alas some of these research endeavours require some gestation or privacy on part of our endeavors.

**Note 3**: Throughout this repo, *miners* will most often refer to Filecoin Storage miners (unless otherwise specified). While we refer to both storage and retrieval miners as miners, strictu sensu, only participation in EC (from storage miners) is mining.

## Table of Contents

- [What is consensus in Filecoin?](#what-is-consensus-in-filecoin?)
- [Consensus Research](#consensus-research)
- [FAQ](#faq)
- [Communication](#communication)
- [License](#license)

## What is consensus in Filecoin?

The state of the Filecoin network is maintained in a blockchain. It is updated as new blocks are mined by a chosen network participant at regular intervals. This "leader" earns FIL for this and is chosen from the set of participants in the Filecoin network with a probability proportional to how much storage this participant is verifiably providing the network. In proceeding thus, block creation
- Provably updates the state of the Filecoin network to reflect network activity (i.e. token transfer and storage proofs);
- Mints new FIL tokens that can be used to buy or sell storage;
- Incentivizes actors to put storage on the network. 

From the perspective of consensus research this can be decomposed into three distinct protocols:

1. **The Filecoin protocol** -- this allows miners to make storage deals with clients, submit and process messages to the state machine and create cryptographically verifiable proofs of storage over data.
1. **Storage Power Consensus (SPC)** -- this generates a power table provably reflecting how much storage participants are providing to the network.
1. **Expected Consensus (EC)** -- invokes leader election to select a miner from a weighted set of participants and ensures chain growth and convergence.

These protocols are meant to be modular and have [interfaces](./research-notes/interfaces.md) which enable us to swap them out or improve them without affecting the rest of the Filecoin stack, for instance updating proofs without changing how SPC functions, or leader election without changing the Filecoin protocol.

## Consensus Research

#### Current state of Filecoin consensus work

To gain familiarity with this subject matter, we refer the reader to:
- [Filecoin Spec](https://github.com/filecoin-project/specs/) -- The spec is the output of these research efforts and reflects our must up-to-date working version of the protocols. The reader may specifically want to look at:
    - [Expected Consensus](https://github.com/filecoin-project/specs/blob/master/expected-consensus.md)
    - [Mining](https://github.com/filecoin-project/specs/blob/master/mining.md) -- the routine that invokes expected consensus.
- [Filecoin Whitepaper](https://filecoin.io/filecoin.pdf) -- We urge readers to pay special attention to Section 6: `Useful Work Consensus`.
- [Power Fault Tolerance Technical Report](https://filecoin.io/power-fault-tolerance.pdf) (PFT) -- Outlining some of the motivations and implications of this work, reframing Byzantine Fault Tolerance as a function of power committed to the network (in our case storage) rather than number of nodes.

#### Current avenues of research

Most of our work on consensus to date focuses on these major endeavours. You can read more about general open problems on the [Filecoin Research repo](https://github.com/filecoin-project/research).

By design, these are meant to be extremely large, open-ended endeavours each of which breaks out into multiple open or completed problems. The endeavours themselves are evergreen sources of enquiry for and beyond Filecoin.

| **Project** | **Description** | **Status** | **Notes** |
| ---- | ---- | ---- | ---- |
| **Formal Treatment of Expected Consensus** | Formal analysis of Expected Consensus' security guarantees | Working On/Collaboration | -[issue](https://github.com/filecoin-project/consensus/issues/19) |
| **EC incentive compatibility** | This broadly refers to EC incentive compatibility and initial parameter setting for the Filecoin blockchain ensuring EC incentive compatibility using simulations or probabilistic proofs. | Working On/Collaboratoin | - Chain convergence <br>- [Weighting](https://github.com/filecoin-project/consensus/issues/27)<br/>- [LBP](https://github.com/filecoin-project/consensus/issues/11)<br>- [Slashing](https://github.com/filecoin-project/consensus/issues/32)<br>- [VDF use](https://github.com/filecoin-project/consensus/issues/25)<br>- [Block time ](https://github.com/filecoin-project/consensus/issues/28)<br>- [Finality](https://github.com/filecoin-project/consensus/issues/29) |
| **Simulate EC Attacks** | Bottoms-up analysis of EC security simulating likely attacks under various proportions of honest/rational/adversarial miners to iterate on protocol design | Working On | -[issue](https://github.com/filecoin-project/consensus/issues/26) |
| **Secret Single Leader Election** | Working out a full construction for SSLE. In spirit similar to cryptographic sortition but guaranteeing a single leader at every round | Collaboration/RFP | - [issue](https://github.com/filecoin-project/research-private/issues/8) <br>- [SSLE RFP](https://github.com/protocol/research-RFPs/blob/master/RFPs/rfp-6-SSLE.md) |
| **Formalizing Power Fault Tolerance (PFT)** | BFT is abstracted in terms of influence over the protocol rather than machines | Working On/Collaboration | - [issue](https://github.com/filecoin-project/consensus/issues/38) |
| **Random beacons and the Filecoin blockchain** | Looking at and beyond the chain for trusted randomness in Filecoin | In Progress (70%) | -[issue](https://github.com/filecoin-project/consensus/issues/24)|

## FAQ

**Why build EC?**

**Q**: Alright, so Filecoin wants a semi-permissionless (or optimally fully-permissionless), robustly reconfigurable consensus protocol that SPC can invoke to do leader election. There are a number of existing proof-of-stake protocol that may be adapted for this purpose. Why roll out our own?

**A**: The answer comes down to a [number of factors](https://github.com/filecoin-project/consensus/issues/13) that boil down to what we have found often happens when trying to adapt theoretical work to real-world security models, including:

- Wanting a secret leader election process (otherwise known as unpredictibility i.e. accounting for DOSing and adaptive attackers)
- Wanting certain liveness guarantees that make MPCs unattractive (use of VDFs)
- Ensuring chain safety under eventual synchrony
- The complexity or partial omissions that we found in other candidate proposals
- Accounting for "rational" miner behaviors rather than simply "honest" or "byzantine" (e.g. [rushing the protocol](research-notes/waiting.md))

With all of that said, it remains important to specify that our work builds upon existing work, notably Snow White and Algorand, and we believe our security analysis will be based on that of Shi and Pass.

**Common misconceptions**

**Q**: Why does EC use tickets for randomness?

**A**: We use tickets for two reasons in the spec as currently laid out:

- Preventing PoST precomputation - we use winning tickets from the previous block as our challenge for a per-slot delay function.
  - Wanted property: "verifiable recency"
- Leader Election - we use the tickets from a past block as a means of secretly and provably checking whether someone has been elected to post the block
  - Wanted property: “verifiable” input on the chain common to all miners

Ultimately, in leader election, we are using tickets in order to approximate a [random beacon](./research-notes/randomness.md), but a promising area of research is to swap this source of randomness out for another on or off-chain source of verifiable randomness.

**Q**: Is block generation (reward and transaction fees) the only way miners will earn FIL?

**A**: No. Miners will also earn FIL through the orders they manage on the network (dealing with clients).
It is interesting to note that a miner must commit storage to the network (and thus appear in the power table) in order to participate in leader election and earn a block reward. This is in fact key to Filecoin's design of a `useful Proof of Work`.
Further, it is worth noting that only storage miners participate in Filecoin consensus. Retrieval miners only earn FIL through deals.

**Q**: Where does collateral come into this?

**A**: This is a direct follow-up to the above question. Because miners earn FIL in two ways (through participation in leader election and in deals), collateral is used in Filecoin to ensure good behavior in both cases. Specifically:

- The Filecoin protocol slashes miners who break a contract (i.e. do not prove they are storing client data during the agreed upon period).

- Expected Consensus slashes miners that sign two distinct blocks in the same slot as a means of speeding up convergence/disincentivizing forks.
- Expected Consensus slashes miners that provably ignore smaller tickets from the tipset, as a means of grinding the chain (and unfairly winning leader election).

The collateral needs for both actions are distinct (in fact EC may not strictly require collateral).

**EC vs SSLE**

**Q**: What is the distinction between EC and SSLE?

**A**: SSLE is an aspirational protocol for finding a single block proposer in leader election. EC is a consensus protocol that includes a block proposer and a way to achieve agreement (PoS Nakamoto consensus) on a particular block.
EC's block proposer function is secret but it is not single.  0 or many blocks could be proposed in a given time slot.
The expected value of proposed leaders per time slot on any chain is 1.
SSLE is an open-problem on which we are actively working as it should greatly simplify the Filecoin consensus construction and lead to faster convergence on the blockchain.

## Communication

- Slack channel: #filecoin-research
- Issues in this repo

## License

The Filecoin Project is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/research/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/research/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT/)
