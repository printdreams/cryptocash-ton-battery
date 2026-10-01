# T1 Spike — W5 (v5R1) gasless preko internal-auth (TON testnet)

Cilj: dokazati da **relayer (poštar) plati gas** za **prazan (0 TON) v5R1 novčanik**, tako što korisnikov potpisani zahtjev isporuči kao **internu poruku** (opcode `auth_signed_internal = 0x73696e74`).

## Zašto se pokreće iz tvog Terminala
Claude-ovo okruženje (i cloud i sandbox) nema izlaz prema TON mreži — dozvoljeni su samo paket-registri. Tvoj obični macOS Terminal ima pun internet, pa se slanje na testnet pokreće tamo.

## Koraci

1. Otvori Terminal i uđi u folder:
   ```bash
   cd ~/Desktop/cryptocash-ton-battery/spike/w5-gasless
   ```

2. Pokreni:
   ```bash
   bash run.sh
   ```
   Prvi put će ispisati **RELAYER** adresu i reći da nije finansirana.

3. Uzmi testne TON-ove s faucet-a na RELAYER adresu (~1 TON je dovoljno):
   - Telegram: **@testgiver_ton_bot** (zalijepiš RELAYER adresu), ili
   - bilo koji TON testnet faucet.

4. Kad stignu coini, ponovo:
   ```bash
   bash run.sh
   ```
   Sad šalje probu i ispisuje rezultat: korisnikov wallet je bio **0 TON**, relayer je platio gas, a na lancu vidiš izvršenu transakciju. Link na explorer je u ispisu.

## Šta dokazuje
- `keys.json` = dvije testne adrese (relayer v4R2 koji plaća, korisnik v5R1 koji ostaje prazan). **Testnet only, ne commita se.**
- Ako korisnikov wallet izvrši akciju a startao je s 0 TON → model „baterija plaća gas za v5" radi, i T7/T12 se mogu graditi na njemu.

> Ovo je throwaway proba (Node). Pravi backend je Go; ovdje samo potvrđujemo mehaniku.
