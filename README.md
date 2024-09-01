# Implementation of Various Byzantine Broadcast Protocols

## Authenticated Broadcast
``` Package: authcbc ```

Authenticated broadcast implements consistent broadcast with a quadratic number of messages and a latency of two message exchanges.

### Overview
Assume a system with ```n``` nodes and a tolerance for ```f``` Byzantine failures.

Intuitively, the sender distributes the request to all parties and expects ```2f+1``` parties to act as witnesses for the request to the others. Every correct party witnesses for the sender’s request by echoing it to all parties; this authenticates the request. When a correct party has received ```2f+1``` such echoes with the same request `m`, then it c-delivers `m`. 

**Note:** In all `upon` clauses of the protocol description below that involve receiving a message, only the first message from each party is considered.

#### Protocol Description

- **upon c-broadcast(m):** (only Ps)
  - Send message `(SEND, m)` to all.

- **upon receiving a message `(SEND, m)` from Ps:**
  - Send message `(ECHO, m)` to all.

- **upon receiving 2f+1 messages `(ECHO, m)`**
  - c-deliver(m)
