import { mnemonicToPrivateKey } from "@ton/crypto";
import {
  TonClient,
  WalletContractV4,
  WalletContractV5R1,
  internal,
  SendMode,
  toNano,
  fromNano,
  beginCell,
} from "@ton/ton";
import { getHttpEndpoint } from "@orbs-network/ton-access";
import fs from "fs";

const TESTNET_GLOBAL_ID = -3;
const keysPath = new URL("./keys.json", import.meta.url).pathname;
const data = JSON.parse(fs.readFileSync(keysPath, "utf8"));

function v5Testnet(publicKey) {
  return WalletContractV5R1.create({
    walletId: {
      networkGlobalId: TESTNET_GLOBAL_ID,
      context: { walletVersion: "v5r1", workchain: 0, subwalletNumber: 0 },
    },
    publicKey,
  });
}

const comment = (t) => beginCell().storeUint(0, 32).storeStringTail(t).endCell();
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const addr = (a) => a.toString({ testOnly: true, bounceable: false });

const relKey = await mnemonicToPrivateKey(data.relayer.mnemonic);
const usrKey = await mnemonicToPrivateKey(data.user.mnemonic);

const endpoint = process.env.TON_ENDPOINT || (await getHttpEndpoint({ network: "testnet" }));
const client = new TonClient({ endpoint, apiKey: process.env.TONCENTER_API_KEY });

const relayer = WalletContractV4.create({ workchain: 0, publicKey: relKey.publicKey });
const user = v5Testnet(usrKey.publicKey);
const relayerC = client.open(relayer);
const userC = client.open(user);

console.log("Endpoint:", endpoint);
console.log("Relayer:", addr(relayer.address));
console.log("User   :", addr(user.address));
console.log("Opcodes v5R1:", JSON.stringify(WalletContractV5R1.OpCodes));

const relBal = await client.getBalance(relayer.address);
const usrBalBefore = await client.getBalance(user.address);
const userDeployedBefore = await client.isContractDeployed(user.address);
console.log("\n--- BEFORE ---");
console.log("Relayer balance:", fromNano(relBal), "TON");
console.log("User balance   :", fromNano(usrBalBefore), "TON  (deployed:", userDeployedBefore + ")");

if (relBal < toNano("0.3")) {
  console.log("\n>>> Relayer nije dovoljno finansiran. Uplati ~1 test TON s faucet-a na ovu adresu pa pokreni ponovo:");
  console.log(">>>", addr(relayer.address));
  process.exit(2);
}

const relSeqno = await relayerC.getSeqno();
const userSeqno = userDeployedBefore ? await userC.getSeqno() : 0;

const signedInternalBody = user.createTransfer({
  seqno: userSeqno,
  authType: "internal",
  secretKey: usrKey.secretKey,
  timeout: Math.floor(Date.now() / 1000) + 300,
  sendMode: SendMode.PAY_GAS_SEPARATELY | SendMode.IGNORE_ERRORS,
  messages: [
    internal({
      to: relayer.address,
      value: toNano("0.02"),
      bounce: false,
      body: comment("gasless-poc-from-zero-balance-v5"),
    }),
  ],
});

console.log(
  "\nSigned-internal body opcode:",
  "0x" + signedInternalBody.beginParse().loadUint(32).toString(16),
  "(auth_signed_internal = 0x" + WalletContractV5R1.OpCodes.auth_signed_internal.toString(16) + ")"
);

await relayerC.sendTransfer({
  seqno: relSeqno,
  secretKey: relKey.secretKey,
  sendMode: SendMode.PAY_GAS_SEPARATELY,
  messages: [
    internal({
      to: user.address,
      value: toNano("0.15"),
      bounce: false,
      init: userDeployedBefore ? undefined : user.init,
      body: signedInternalBody,
    }),
  ],
});

console.log("\nRelayer je poslao internu poruku. Čekam da korisnikov v5 wallet izvrši akciju...");

for (let i = 0; i < 30; i++) {
  await sleep(3000);
  const deployed = await client.isContractDeployed(user.address);
  const s = deployed ? await userC.getSeqno().catch(() => 0) : 0;
  if (deployed && s >= userSeqno + 1) break;
  process.stdout.write(".");
}

const usrBalAfter = await client.getBalance(user.address);
console.log("\n\n--- AFTER ---");
console.log("User deployed  :", await client.isContractDeployed(user.address));
console.log("User seqno     :", await userC.getSeqno().catch(() => "n/a"));
console.log("User balance   :", fromNano(usrBalAfter), "TON");

const txs = await client.getTransactions(user.address, { limit: 3 });
console.log("\n--- Zadnje transakcije korisnikovog wallet-a (dokaz na lancu) ---");
for (const tx of txs) {
  const lt = tx.lt.toString();
  const hash = tx.hash().toString("hex");
  const inOp = tx.inMessage?.body ? tryOp(tx.inMessage.body) : "n/a";
  console.log(`lt=${lt} hash=${hash} inMsgOp=${inOp} outMsgs=${tx.outMessagesCount}`);
}
console.log("\nExplorer:", `https://testnet.tonviewer.com/${addr(user.address)}`);

function tryOp(body) {
  try {
    const s = body.beginParse();
    if (s.remainingBits >= 32) return "0x" + s.loadUint(32).toString(16);
  } catch {}
  return "n/a";
}
