# LGT DRM / auth handshake — protocol map

Reverse-engineered from `하이브리드` `binary.mod` (ARM/Thumb, IDA Hex-Rays) plus
runtime tracing in the aram-core raptor runtime, 2026-08-17/18. Addresses are
`하이브리드`-specific; the API (ordinals, memory layout roles) is the shared LGT
platform contract.

## Entry / UI flow

1. Intro animation.
2. `이용안내` screen — any key.
3. `게임 실행을 위한 인증을 시작합니다 (통화료 5원 미만)` — `1.예 / 2.아니오`.
4. On `예`: `서버 접속중...` while the DRM handshake runs. Never completes
   without a server → hang.

## The auth object

- A DRM/auth **class**: vtable at `0x8ddc8`, singleton instance at
  `session = *(u32*)0x1400058`.
- vtable methods (index → fn): `[5] sub_5AC4` (event-bit reader),
  `[8] sub_3A430` (protocol tick). The game loop calls the tick each frame.
- Key session fields (offsets from `session`):
  - `+0x56C`, `+0x570` — connection handles.
  - `+0x574` — **auth state** (see below).
  - `+0x578 / +0x57C / +0x580` — parsed response fields (result code + tokens).
  - `+0x588` — when nonzero in state 300, `state = *(session+0x588)` (jump).
  - `+0x589`, `+0x58A` — extra gate flags (must be 0 for the success paths).
  - `+0x58E + N` — **event-bit array**; `sub_5AC4(session, N)` returns
    `*(byte*)(session + 0x58E + N) != 0`.

## Network primitives (ordinals)

The raptor veneers `sub_8C854/858/85C` are `BX R3/R4/R5` and are used for BOTH
guest indirect calls AND import trampolines installed in the executable
`0x1400000` RW region. `0x1408fdc` is the **ordinal-106 trampoline** (not a data
buffer — an early misread).

- **106**(`r0=0xa600` event-class, `r1=0`, `r2=0`) — poll / get-state.
- **238**(`r0=<106 result>`, `r1=0xa600`, `r2=0`) — receive / process; returns a
  status byte. Wrapper `sub_2F684` calls 106 then 238.
- `0xa600` is a **constant event-class id**, not a created handle (no connect
  ordinal precedes it).
- Changing the **return values** of 106/238 has no effect — the game reads
  server-populated **guest memory** (session arrays + event bytes), not the
  return. So the backend must write session state, not just return codes.
- A "send" primitive has not yet been positively identified; the request side
  still needs observation (that is the next RE step).

## State machine (`sub_3A430`, the tick)

Reads `state = session+0x574`, switches:

- `case -300 (0xFFFFFED4)`: `state = 0`, poll(9). (connect/retry)
- `case 300 (0x12C)`: if `session+0x588 != 0` then `state = *(session+0x588)`;
  poll(3). (leaves the connecting state only when +0x588 is set)
- `case -74 (0xFFFFFFB6)`: poll(6).
- `default` (incl. 0, 21, 37): `state = receive()`.
  - `receive` = guest fn `sub_5764C`; reads a short from a session array via
    the copy import `0x1408f5c`. Returns `-300` while no response.
  - if `state == 300`: check event bits via `sub_5AC4`:
    - `ev1 & ev2 & ev3 & !ev4 & session[0x589]==0` → **state = 21**
    - `!ev10 & ev7 & ev8 & ev9 & session[0x58A]==0` → **state = 37**
    - else keep waiting.
  - if `state >= 0 && state != 300`: **success branch** — read response fields
    (byte `sub_57611`, short `sub_5764D`) into `+0x578/+0x57C/+0x580`, then
    poll(result_code).
  - if `state < 0`: error (`-10` → code 6, else code 4).

Observed at runtime: the game sits in the **`-300` connect/retry loop** because
`receive` never returns anything but -300.

## What a working backend must do (hypothesis)

Drive the handshake to a terminal "authorized" state by populating session
memory across successive ticks:
1. Make `receive` stop returning -300 (advance to state 300).
2. Set event bits `session[0x58E+1..3]`, clear `+4`, `+0x589` → state 21;
   OR set `+0x588` to the next state.
3. Supply valid response fields for the success branch.
4. Continue until the auth object signals "done" and the game loop switches off
   the `서버 접속중` screen.

The exact state sequence + field values that constitute "authorized" are the
remaining unknown; derive them iteratively with the game as oracle (feed a
candidate response, observe whether `+0x574` leaves -300 and the screen
advances).

## Open questions

- Full list of network ordinals (is there a connect/send besides 106/238?).
- The `receive` session-array format and the "authorized" result code.
- The terminal state that makes the game leave the auth screen and launch.
