# Implementation of Various Byzantine Broadcast Primitives

## How to run a simulation

```bash
git clone git@github.com:ShivaanshK/Lamport-SMR.git
cd Lamport-SMR
```
```bash
# In the first shell
go run main.go -pid=0 -type={BROADCAST_TYPE}
# In the second shell
go run main.go -pid=1 -type={BROADCAST_TYPE}
# In the third shell
go run main.go -pid=2 -type={BROADCAST_TYPE}
# In the fourth shell
go run main.go -pid=3 -type={BROADCAST_TYPE}
```

```BROADCAST_TYPE``` is defined as an enumeration:
```AUTHENTICATED_BROADCAST``` = 0
```BRACHA_BROADCAST``` = 1
---

## Consistent Broadcast (CBC)

### Properties

1. **Validity**  
   If a correct sender \(P_s\) c-broadcasts message `m`, then all correct parties eventually c-deliver `m`.

2. **Consistency**  
   If a correct party c-delivers message `m`, and another correct party c-delivers message `m'`, then \(m = m'\).

3. **Integrity**  
   Every correct party c-delivers at most one message. Moreover, if the sender \(P_s\) is correct, then the message was previously c-broadcast by \(P_s\).

---

### Authenticated Broadcast

**Package:** `authenticated`

Authenticated broadcast implements **consistent broadcast** with a quadratic number of messages and a latency of two message exchanges.

#### Overview

Assume a system with `n` nodes and a tolerance for `f` Byzantine failures.

The protocol involves a sender distributing the request to all parties, expecting `2f + 1` parties to act as witnesses for the request. Every correct party authenticates the request by echoing it to all other parties. When a correct party receives `2f + 1` such echoes for the same request `m`, it c-delivers `m`.

#### Protocol Description

**Note**: In all `upon` clauses below that involve receiving a message, only the first message from each party is considered. This algorithm assumes that `n > 3f.`

1. **upon c-broadcast(m):** (only for the sender \(P_s\))  
   - The sender \(P_s\) sends the message `(SEND, m)` to all other nodes.

2. **upon receiving a message `(SEND, m)` from \(P_s\):**  
   - Each party sends the message `(ECHO, m)` to all other nodes.

3. **upon receiving `2f + 1` messages `(ECHO, m)`:**  
   - The party c-delivers `m` (i.e., considers the message `m` to be consistently delivered).

---

## Reliable Broadcast (RBC)

### Properties

1. **Validity**  
   If a correct sender \(P_s\) r-broadcasts message `m`, then all correct parties eventually r-deliver `m`.

2. **Consistency**  
   If a correct party r-delivers message `m`, and another correct party r-delivers message `m'`, then \(m = m'\).

3. **Integrity**  
   Every correct party r-delivers at most one message. Moreover, if the sender \(P_s\) is correct, then the message was previously r-broadcast by \(P_s\).

4. **Totality**  
   If some correct party r-delivers a request, then all correct parties eventually r-deliver a request.

---

### Bracha Broadcast

**Package:** `bracha`

Bracha broadcast is a classical implementation of reliable broadcast that uses two rounds of message exchanges among all parties. The protocol is resilient to Byzantine failures and ensures that all correct parties r-deliver the same message. The process includes sending, echoing, and confirming readiness before delivering the message.

#### Overview

Assume a system with `n` nodes and a tolerance for `f` Byzantine failures.

After receiving the request from the sender, every party echoes the request to all other parties. Once a party receives `2f + 1` echo messages from other parties, it indicates to others that it is ready to deliver the message. Finally, when a party receives enough ready messages, it r-delivers the request.

#### Protocol Description

**Note**: In all `upon` clauses that involve receiving a message, only the first message from each party is considered. This algorithm assumes that `n > 3f.`

1. **upon r-broadcast(m):** (only for the sender \(P_s\))  
   - The sender \(P_s\) sends the message `(SEND, m)` to all other nodes.

2. **upon receiving a message `(SEND, m)` from \(P_s\):**  
   - Each party sends the message `(ECHO, m)` to all other nodes.

3. **upon receiving `2f + 1` messages `(ECHO, m)` and not having sent a `READY` message:**  
   - The party sends the message `(READY, m)` to all other nodes.

4. **upon receiving `f + 1` messages `(READY, m)` and not having sent a `READY` message:**  
   - The party sends the message `(READY, m)` to all other nodes.

5. **upon receiving `2f + 1` messages `(READY, m)`:**  
   - The party r-delivers `m`.

