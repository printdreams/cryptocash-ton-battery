import { mnemonicNew, mnemonicToPrivateKey } from "@ton/crypto";
import { WalletContractV4, WalletContractV5R1 } from "@ton/ton";
import fs from "fs";

const TESTNET_GLOBAL_ID = -3;
const keysPath = new URL("./keys.json", import.meta.url).pathname;
const seedPath = new URL("./relayer.seed", import.meta.url).pathname;

function v5Testnet(publicKey) {
  return WalletContractV5R1.create({
    walletId: {
      networkGlobalId: TESTNET_GLOBAL_ID,
      context: { walletVersion: "v5r1", workchain: 0, subwalletNumber: 0 },
    },
    publicKey,
  });
}

const existing = fs.existsSync(keysPath) ? JSON.parse(fs.readFileSync(keysPath, "utf8")) : null;

let relMnemonic;
let relSource;
if (fs.existsSync(seedPath)) {
  relMnemonic = fs.readFileSync(seedPath, "utf8").trim().split(/\s+/);
  if (relMnemonic.length !== 24) {
    console.error(`relayer.seed mora imati tačno 24 riječi, a ima ${relMnemonic.length}.`);
    process.exit(1);
  }
  relSource = "relayer.seed (tvoj wallet)";
} else if (existing?.relayer?.mnemonic) {
  relMnemonic = existing.relayer.mnemonic;
  relSource = "keys.json (ranije)";
} else {
  relMnemonic = await mnemonicNew();
  relSource = "novo generisan (throwaway)";
}

const usrMnemonic = existing?.user?.mnemonic || (await mnemonicNew());

const relKey = await mnemonicToPrivateKey(relMnemonic);
const usrKey = await mnemonicToPrivateKey(usrMnemonic);
const relayer = WalletContractV4.create({ workchain: 0, publicKey: relKey.publicKey });
const user = v5Testnet(usrKey.publicKey);
const relAddr = relayer.address.toString({ testOnly: true, bounceable: false });
const usrAddr = user.address.toString({ testOnly: true, bounceable: false });

fs.writeFileSync(
  keysPath,
  JSON.stringify(
    {
      network: "testnet",
      relayer: { type: "v4R2", mnemonic: relMnemonic, address: relAddr },
      user: { type: "v5R1", mnemonic: usrMnemonic, address: usrAddr },
    },
    null,
    2
  )
);

console.log("Relayer izvor:", relSource);
console.log("RELAYER (v4R2) testnet adresa:", relAddr, "  <-- OVU adresu puniš test-TON-om");
console.log("USER    (v5R1) testnet adresa:", usrAddr, "  (ostaje prazan, 0)");
