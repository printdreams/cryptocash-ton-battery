#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

echo "==> Instaliram zavisnosti (samo prvi put)"
npm install --silent

echo "==> Generišem/čitam testne ključeve"
node keys.mjs

echo
echo "==> Pokrećem probu (ako relayer nije finansiran, reći će ti koju adresu da uplatiš)"
node gasless.mjs
