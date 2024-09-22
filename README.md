# Implementation of Various Byzantine Broadcast Primitives

## Run a Simulation

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

The ```BROADCAST_TYPE``` enumeration is defined as follows:
1. ```AUTHENTICATED_BROADCAST``` = 0
2. ```ECHO_BROADCAST``` = 1
3. ```BRACHA_BROADCAST``` = 2

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

### Echo Broadcast

**Package:** `echo`

The Echo Broadcast protocol is another implementation of **consistent broadcast**, which leverages digital signatures to achieve lower message and communication complexity compared to the authenticated broadcast protocol. However, this comes at the cost of increased latency and reliance on the sender's liveness. Instead of relying on witnesses to authenticate a request through message exchanges, as in the authenticated broadcast, the echo broadcast uses signed statements disseminated by the sender.

#### Overview

Assume a system with `n` nodes and a tolerance for `f` Byzantine failures.

The echo broadcast protocol involves a sender distributing the request to all parties, expecting a majority of parties (specifically, `2f+1`, where `f` is the number of tolerated failures) to act as witnesses by signing and sending back an `ECHO` message. Once the sender receives enough valid signatures, it sends a `FINAL` message to all parties, which leads to the consistent delivery of the message.

#### Protocol Description

**Note**: In all `upon` clauses below that involve receiving a message, only the first message from each party is considered. This algorithm assumes that `n > 3f`.

1. **upon c-broadcast(m):** (only for the sender \(P_s\))  
   - The sender \(P_s\) sends the message `(SEND, m)` to all other nodes.

2. **upon receiving a message `(SEND, m)` from \(P_s\):**  
   - Each party creates a digital signature for the message `m`:
     ```go 
     σ := sign_i("ECHO" || s || m)
     ```
     - The party sends the message `(ECHO, m, σ)` to the sender \(P_s\).

3. **upon receiving `2f+1` messages `(ECHO, m, σ_j)` with valid signatures σ_j:** (only for the sender \(P_s\))  
   - The sender verifies the received signatures and, once the threshold is reached, forms a list of all received signatures \(Σ\).
   - The sender sends the message `(FINAL, m, Σ)` to all other nodes.

4. **upon receiving a message `(FINAL, m, Σ)` from \(P_s\) with `2f+1` valid signatures in Σ:**  
   - The party c-delivers `m` (i.e., considers the message `m` to be consistently delivered).

#### Summary

The Echo Broadcast protocol achieves consistent broadcast using digital signatures, ensuring that messages are delivered consistently to all correct parties. Its communication complexity is \(O(n)\), and its latency is three message exchanges. Compared to authenticated broadcast, it achieves lower communication complexity but at the cost of 1 more message exchange.


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

