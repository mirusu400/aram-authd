# aram-authd

Pure-Go emulator for the **LGT carrier DRM / server-authentication protocol**
that many 엔소니-published LGT WIPI-C (raptor) games require before they will
start.

Titles like `제노니아1`, `하이브리드`, and others show

```
이용안내  →  게임 실행을 위한 인증을 시작합니다  →  서버 접속중...
```

and then poll the LGT network/event API (ordinals **106 / 238**) waiting for an
"authorized" response from a carrier server that has been offline for ~15 years.
With no server, the games hang forever at `서버 접속중`.

`aram-authd` synthesizes the server side of that handshake in-process so the
games authenticate locally and reach gameplay. It is a **pure-Go library**
(not a live network service) so `aram-test` replay stays deterministic.

## Where it fits

```
aram-frontend  <-  aram-emu  ->  aram-core        (existing dependency direction)
                      |
                      +------->  aram-authd        (injected as the raptor NetBackend)
```

- **aram-core** defines a `NetBackend` interface and routes the raptor network
  ordinals (106/238, plus any related connect/send/recv) to it. The default
  backend is a no-op, preserving today's behavior.
- **aram-authd** implements `NetBackend` with the LGT DRM handshake.
- **aram-emu** (the composition root) injects `aram-authd` into the machine, so
  `aram-core` keeps no hard dependency on it and stays Android/arm64 pure-Go.

## Status

Reverse-engineering the protocol. See [`docs/protocol.md`](docs/protocol.md) for
the current map. The development loop uses the running game as an oracle:
bridge 106/238 → observe the guest's requests → craft a response → watch the
auth state machine advance past `-300` → iterate.

Unblocks: aram-core issues #52 (하이브리드), #49 (제노니아1 sound), #36 (제노니아 auth).
