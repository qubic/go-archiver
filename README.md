# The qubic archiver service

> The archiver service's purpose is to store and make available data regardless of the current epoch.

## High level description:
The archive system consists of two services:
- `qubic-archiver` - the archiver processor and HTTP server that provides rpc endpoints to query the archiver
- `qubic-nodes` - a service responsible with providing information regarding reliable nodes and the max tick of the
  network

## IMPORTANT

> [!WARNING]  
> This version of archiver is **INCOMPATIBLE** by default with versions **v0.x.x**, as it features database compression and a different format for quorum data.  
> If you wish to enable backwards compatibility, please set `STORE_COMPRESSION_TYPE` to `Snappy` and `STORE_SAVE_FULL_QUORUM_DATA` to `true`.  
> Archiver **DOES NOT** migrate the database to the new format by itself, and **MAY BREAK** your existing information, if not migrated correctly.  
> For a migration tool, please see the [Archiver DB Migrator](https://github.com/qubic/archiver-db-migrator), and make sure to back up your data!  

Before starting the system, open the `docker-compose.yml` file and make sure that you have a reliable peer list setup
for the `qubic-nodes` service.
This can be configured using the `QUBIC_NODES_QUBIC_PEER_LIST` environment variable.

## Other optional configuration parameters for qubic-archiver can be specified as env variable by adding them to docker compose:

```bash
  $QUBIC_ARCHIVER_SERVER_READ_TIMEOUT                        <duration>  (default: 5s)
  $QUBIC_ARCHIVER_SERVER_WRITE_TIMEOUT                       <duration>  (default: 5s)
  $QUBIC_ARCHIVER_SERVER_SHUTDOWN_TIMEOUT                    <duration>  (default: 5s)
  $QUBIC_ARCHIVER_SERVER_HTTP_HOST                           <string>    (default: 0.0.0.0:8000)
  $QUBIC_ARCHIVER_SERVER_GRPC_HOST                           <string>    (default: 0.0.0.0:8001)
  $QUBIC_ARCHIVER_SERVER_NODE_SYNC_THRESHOLD                 <int>       (default: 3)
  $QUBIC_ARCHIVER_SERVER_CHAIN_TICK_FETCH_URL                <string>    (default: http://127.0.0.1:8080/max-tick)
  
  $QUBIC_ARCHIVER_POOL_NODE_FETCHER_URL                      <string>    (default: http://127.0.0.1:8080/status)
  $QUBIC_ARCHIVER_POOL_NODE_FETCHER_TIMEOUT                  <duration>  (default: 2s)
  $QUBIC_ARCHIVER_POOL_INITIAL_CAP                           <int>       (default: 5)
  $QUBIC_ARCHIVER_POOL_MAX_IDLE                              <int>       (default: 20)
  $QUBIC_ARCHIVER_POOL_MAX_CAP                               <int>       (default: 30)
  $QUBIC_ARCHIVER_POOL_IDLE_TIMEOUT                          <duration>  (default: 15s)
  
  $QUBIC_ARCHIVER_QUBIC_NODE_PORT                            <string>    (default: 21841)
  $QUBIC_ARCHIVER_QUBIC_STORAGE_FOLDER                       <string>    (default: store)
  $QUBIC_ARCHIVER_QUBIC_PROCESS_TICK_TIMEOUT                 <duration>  (default: 5s)
  $QUBIC_ARCHIVER_STORE_COMPRESSION_TYPE                     <string>    (default: Zstd)
  $QUBIC_ARCHIVER_STORE_SAVE_FULL_VOTE_DATA                  <bool>      (default: false)
```

## Run with docker-compose:

```bash
$ docker-compose up -d
```

## Available endpoints:

### Instance information

#### /status
Provides information regarding the status of the archiver instance.

```shell
curl http://127.0.0.1:8001/status
```
```json
{
  "lastProcessedTick":{
    "tickNumber":13683006,
    "epoch":107
  },
  "lastProcessedTicksPerEpoch":{
    "106":13548510,
    "107":13683006
  },
  "skippedTicks":[
    {
      "startTick":1,
      "endTick":13547629
    },
    {
      "startTick":13548511,
      "endTick":13679999
    }
  ],
  "processedTickIntervalsPerEpoch":[
    {
      "epoch":106,
      "intervals":[
        {
          "initialProcessedTick":13547630,
          "lastProcessedTick":13548510
        }
      ]
    },
    {
      "epoch":107,
      "intervals":[
        {
          "initialProcessedTick":13680000,
          "lastProcessedTick":13683006
        }
      ]
    }
  ]
}
```

#### /healthcheck
Mainly used by the load-balancer to decide if the instance should be added to the balancing rotation based on if it's up-to-date with the network or not.

```shell
curl http://127.0.0.1:8001/healthcheck
```
```json
{
  "code":13,
  "message":"processor is behind node by 7684 ticks",
  "details":[]
}
```

#### /latestTick
Returns the number of the latest tick processed by the archiver instance.

```shell
curl http://127.0.0.1:8001/latestTick
```
```json
{
  "latestTick":13690806
}
```

***

### Tick related endpoints

#### /ticks/{tick_number}/tick-data
Returns the tick information for the given tick.
```shell
curl http://127.0.0.1:8001/ticks/13683397/tick-data
```
```json
{
  "tickData":{
    "computorIndex":481,
    "epoch":107,
    "tickNumber":13683397,
    "timestamp":"1714593920000",
    "varStruct":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
    "timeLock":"1BAUdVy8iM0d9LYFK2/WACnR+Fvn7cPY3sLgnt2W/Es=",
    "transactionIds":[
      "xgniuxigsnbeifvkithkcgnvxhmglgkppscwupescgwoqljxdecekhueutfn",
      "ilhefvlxgzpzmdfiixupcusbgyihxxqfomdvumwfngkuugagyutjudhfhobd",
      "jvpovfnhpiijehctdtsivngewabbzwxinjijwmeiihuebraaheokpzdayhsb"
    ],
    "contractFees":[],
    "signatureHex":"74b77a2f61b363bc0f2ebeda39b5255c8b5ffb767f571a32603ace8b4f47c23f531018bb4efac3f5a9c3f7896c08d04e4149319a666df25d0bed6c6cfbed0000"
  }
}
```

#### /ticks/{tick_number}/quorum-tick-data
Returns the quorum tick information for the given tick.
```shell
curl http://127.0.0.1:8001/ticks/13683397/quorum-tick-data
```
```json
{
  "quorumTickData":{
    "quorumTickStructure":{
      "epoch":107,
      "tickNumber":13683397,
      "timestamp":"1714593914000",
      "prevResourceTestingDigestHex":"42359ba71f11311f",
      "prevSpectrumDigestHex":"f635766cbfe80a021e18d03a87fc8304e7f8cdd25e56f8147e2b97e21ea38161",
      "prevUniverseDigestHex":"52da6fe4a4da50a34582f11b53094ebdfa25087614a5dc6e5e15f0f418804102",
      "prevComputerDigestHex":"9ea25c5c9fa90e55d68cca1d3c78183d5b91bbac82c96de5400e267a16223bd2",
      "txDigestHex":"09ab681628b3e0263c444c2edbce51adb22c0e38e81cbfedd7c08abab2d6c67f"
    },
    "quorumDiffPerComputor":{
      "0":{
        "saltedResourceTestingDigestHex":"0ee4483f607b4865",
        "saltedSpectrumDigestHex":"41d5452637b88cba3b9b44c320ff958f26f7898864752c82ee42ebcacedb6e40",
        "saltedUniverseDigestHex":"751da9c5e2d9e9a74dae2e6552ffaad249ac0eae728a0c940d1cdb3d3e1c8fcd",
        "saltedComputerDigestHex":"755957976e92e0bf65e5537ec4398867f1afabfc777ab4f888e3ccecd6ff787a",
        "expectedNextTickTxDigestHex":"b0f4b3415740c927d351c528355ab13bd2d0640395b5c0eae380e7b93ca5872d",
        "signatureHex":"36ac100871d93c0719370fc75f034e72780f7c4fc21605fe98ec69229a7df2d1fdc1f65aa3bcc74c927bc9296400914bc8909677b2f1e26643764718fcc40100"
      },
      "1":{
        "saltedResourceTestingDigestHex":"d0bfef154b6bbc28",
        "saltedSpectrumDigestHex":"43d1c210e61c3481df7bc65a5456d089be0932ddabac0690ba5a4df918561ef0",
        "saltedUniverseDigestHex":"86bff2d98c8a04517cfe92d662f2185640a3af0a6e125b7e9353119f57b96e3a",
        "saltedComputerDigestHex":"46d7e8504caa635789a8c014e2270db92e8f996851637eca11ddf7f533e76b21",
        "expectedNextTickTxDigestHex":"b0f4b3415740c927d351c528355ab13bd2d0640395b5c0eae380e7b93ca5872d",
        "signatureHex":"c47b58fc51247bb4d223d922e589771956103280c82ccb0927da464bdd0861791d788109c0bdee63dc94553215fc21d5db994eb1e4c1ec268a8d7663fd8d2500"
      },
//      ...
      "675":{
        "saltedResourceTestingDigestHex":"fd730f13a8973bd3",
        "saltedSpectrumDigestHex":"ae20699498a6231169d936fd749a5f8ce5c7608ee8bb019824ff984c7e7f0321",
        "saltedUniverseDigestHex":"9cb63a800d5c8bce5933c82c322205db7a6be9d0389512193b33166efcb0ac61",
        "saltedComputerDigestHex":"816131300e1e13e7e9bf9fc5831653764ea9fb2a0bd7e01570db2e049a3f2caa",
        "expectedNextTickTxDigestHex":"b0f4b3415740c927d351c528355ab13bd2d0640395b5c0eae380e7b93ca5872d",
        "signatureHex":"c4f4ac87c1edf351e60bb545d15a457d7d803646f1c324d9c2429213c326ace736181b1135f718b97a52649a9c6abb43b3774cb0049bf2f959d5cfa8e08c0300"
      }
    }
  }
}
```

#### /ticks/{tick_number}/transactions
Returns the full list of transactions for the given tick.  
> Note that this will include **ALL** transactions, **approved or not**.

```shell
curl http://127.0.0.1:8001/ticks/13683397/transactions
```
```json
{
  "transactions":[
    {
      "sourceId":"IAHIRNYARPTESFWWKHWBIICNUEYCIGFMXOGOOBNNBAKSEEGFDWUHPNDHJQUK",
      "destId":"AFZPUAIYVPNUYGJRQVLUKOPPVLHAZQTGLYAAUUNBXFTVTAMSBKQBLEIEPCVJ",
      "amount":"0",
      "tickNumber":13683397,
      "inputType":0,
      "inputSize":32,
      "inputHex":"716c692d637564616b008dd1814005e586d28cbf0687c5d380817ee3f5494304",
      "signatureHex":"814d9e9ccb01766b0885d0592e6c39054c9311d13971be51e3a798c9f3ec080c6f280e4b6519042ee4a9be0c8d9b23127c9ee788282e7578cc8d52dd351c1d00",
      "txId":"xgniuxigsnbeifvkithkcgnvxhmglgkppscwupescgwoqljxdecekhueutfn"
    },
    {
      "sourceId":"UHDZJOGELURMKBEQUSMNBMWAVPHDADXMZPCOLTFAUGQZVELHGUYRCPADDBEF",
      "destId":"AFZPUAIYVPNUYGJRQVLUKOPPVLHAZQTGLYAAUUNBXFTVTAMSBKQBLEIEPCVJ",
      "amount":"0",
      "tickNumber":13683397,
      "inputType":0,
      "inputSize":32,
      "inputHex":"716c692d637564616b0197b23003bc90c9b26688c3329109deba9654162b5c12",
      "signatureHex":"8b165c45532b2a2ef52da2c7bb78030ceacb4154c107297dc6f0f2ccb9da5dd2b602b8578ba391a916c1fefbcb31115b6345456fac868a188cfdeb5530b21100",
      "txId":"ilhefvlxgzpzmdfiixupcusbgyihxxqfomdvumwfngkuugagyutjudhfhobd"
    },
    {
      "sourceId":"MYVLWZGGVHALIEVZQXCEVQJXNEUAYULLEMBMFTNORGJGESCMQTPCVOHBBBTI",
      "destId":"AFZPUAIYVPNUYGJRQVLUKOPPVLHAZQTGLYAAUUNBXFTVTAMSBKQBLEIEPCVJ",
      "amount":"0",
      "tickNumber":13683397,
      "inputType":0,
      "inputSize":32,
      "inputHex":"716c692d637564616b01c8af176f1f3b2ec5b8e53af9215b86d139dbc835003d",
      "signatureHex":"7bea63bcf5fe31c670ec1feee93c943c5e8c4f12fc58f69cfe50b59f0a92dd3347b9ab2797b8184441a4be885ac4579a1000fdf429b0a6dd540bf0cff12a0300",
      "txId":"jvpovfnhpiijehctdtsivngewabbzwxinjijwmeiihuebraaheokpzdayhsb"
    }
  ]
}
```

#### /ticks/{tick_number}/transfer-transactions
Returns the list of transfer transactions for the given tick.
> Note that there is a difference between **transfer** transactions and **mining** transactions.

```shell
curl http://127.0.0.1:8001/ticks/13686173/transfer-transactions
```
```json
{
  "transactions": [
    {
      "sourceId": "FNXHQOKFGKMZUGWQHLTNOPMIGXQAQZWYUPAGXMGAKAASHCVNGUPHEMJHUSOK",
      "destId": "IZTNWDKXSFULQADTOLTMLUPHSCFCXLOJMQOUHPBSRGQZMMXZCJYQFTRDOGRE",
      "amount": "45832157",
      "tickNumber": 13686173,
      "inputType": 0,
      "inputSize": 0,
      "inputHex": "",
      "signatureHex": "8b897f20911d4df01c9faa56782760173f95e1da6f22bbbbae13a519904356b25db5eb736a27056d34f29d09b8fc847142703c0deacedb143b759acd42be0b00",
      "txId": "whhhorfprtqayfygoqwoyqbpnajdfwocnyunwwvjqgykcqpnmphrbozdxscj"
    }
  ]
}
```

#### /ticks/{tick_number}/approved-transactions
Returns the list of all approved transactions for the given tick.

```shell
curl http://127.0.0.1:8001/ticks/13686387/approved-transactions
```
```json
{
  "approvedTransactions": [
    {
      "sourceId": "ARALPBGBRNORYBDFRWKQSLENOELBMFJWOFKBRQJNXDXTRZPYGGFKSADAXJON",
      "destId": "NLRQDYJUXUDLTEMGPZSBWAABQTIAYZCELAOZIAPBTGTRMGFPTTEBALRAYPPN",
      "amount": "24427392",
      "tickNumber": 13686387,
      "inputType": 0,
      "inputSize": 0,
      "inputHex": "",
      "signatureHex": "526370cd218a33d53ff17d1de4194158e713222ebee88e53d26c73872d1946fcaae42ec83be1a4b84dbf14bc5458ed462ab43ccc63c4446554341e782c541700",
      "txId": "ktwllcxqbvlrffrbweestshxqxbhpulqwdnvljssmcuzuefuzcwufedgmkya"
    }
  ]
}
```

#### /ticks/{tick_number}/chain-hash
Returns the hash of the given processed tick. This is mainly used to compare archiver instances and verify they process ticks the same.

```shell
curl http://127.0.0.1:8001/ticks/13686387/chain-tick
```
```json
{
  "hexDigest": "cf8914a346c217036a71d92f4e81c123aeea7a2d187f4123932e98326360c101"
}
```
***

### Transaction related endpoints

#### /transactions/{tx_id}
Returns the transaction information for the given transaction id.

```shell
curl http://127.0.0.1:8001/transactions/ktwllcxqbvlrffrbweestshxqxbhpulqwdnvljssmcuzuefuzcwufedgmkya
```
```json
{
  "transaction": {
    "sourceId": "ARALPBGBRNORYBDFRWKQSLENOELBMFJWOFKBRQJNXDXTRZPYGGFKSADAXJON",
    "destId": "NLRQDYJUXUDLTEMGPZSBWAABQTIAYZCELAOZIAPBTGTRMGFPTTEBALRAYPPN",
    "amount": "24427392",
    "tickNumber": 13686387,
    "inputType": 0,
    "inputSize": 0,
    "inputHex": "",
    "signatureHex": "526370cd218a33d53ff17d1de4194158e713222ebee88e53d26c73872d1946fcaae42ec83be1a4b84dbf14bc5458ed462ab43ccc63c4446554341e782c541700",
    "txId": "ktwllcxqbvlrffrbweestshxqxbhpulqwdnvljssmcuzuefuzcwufedgmkya"
  }
}
```

#### /tx-status/{tx_id}
Returns the status of the given transaction.

```shell
curl http://127.0.0.1:8001/tx-status/ktwllcxqbvlrffrbweestshxqxbhpulqwdnvljssmcuzuefuzcwufedgmkya
```
```json
{
  "transactionStatus": {
    "txId": "ktwllcxqbvlrffrbweestshxqxbhpulqwdnvljssmcuzuefuzcwufedgmkya",
    "moneyFlew": true
  }
}
```

#### /identities/{identity}/transfer-transactions
Returns the list of **transfer** transactions for the given identity.

```shell
curl http://127.0.0.1:8001/identities/ARALPBGBRNORYBDFRWKQSLENOELBMFJWOFKBRQJNXDXTRZPYGGFKSADAXJON/transfer-transactions?start_tick=13686000&end_tick=13686400
```
```json
{
  "transferTransactionsPerTick": [
    {
      "tickNumber": 13686387,
      "identity": "ARALPBGBRNORYBDFRWKQSLENOELBMFJWOFKBRQJNXDXTRZPYGGFKSADAXJON",
      "transactions": [
        {
          "sourceId": "ARALPBGBRNORYBDFRWKQSLENOELBMFJWOFKBRQJNXDXTRZPYGGFKSADAXJON",
          "destId": "NLRQDYJUXUDLTEMGPZSBWAABQTIAYZCELAOZIAPBTGTRMGFPTTEBALRAYPPN",
          "amount": "24427392",
          "tickNumber": 13686387,
          "inputType": 0,
          "inputSize": 0,
          "inputHex": "",
          "signatureHex": "526370cd218a33d53ff17d1de4194158e713222ebee88e53d26c73872d1946fcaae42ec83be1a4b84dbf14bc5458ed462ab43ccc63c4446554341e782c541700",
          "txId": "ktwllcxqbvlrffrbweestshxqxbhpulqwdnvljssmcuzuefuzcwufedgmkya"
        }
      ]
    }
  ]
}
```

***

### Epoch related endpoints

#### /epochs/{epoch}/computors
Returns the list of computors for the given epoch

```shell
curl http://127.0.0.1:8001/epochs/107/computors
```
```json
{
  "computors": {
    "epoch": 107,
    "identities": [
      "OTSWYJAUPATSAHMJBEYVRCPZXJPAYQOEWWYEZKUTWELLTLKHCPPGWXBCXVVH",
      "AUHFHHNWLWPXEEXRCAKDWUCYRDRAZNRDNISHEDZYAFYJOUZZKYFMMWFAAFXM",
      "YQZPLDQLMCWDSDETFBBABRXGNRTCZSSXFYSGBPFCZFNSSQZDJVDQYUPDNYIM",
      "RGGNEEZYXQYTYFNFTLQYZKNNFMSCTBRSNZJIQGCXKAVVELCXQQQRMAKDDGOA",
      "JKUYHUUTSYESTATAZVGPQCFFCHHACXLEBXJWUBNBBAFGPSRLNFLNYRPGXEHO",
      "HDPWVFRQBPZREDVBVHYJLMLDACFDLPCLKGJOMFETCGEHMPVNTXVOVVHCNATJ",
      "SPECJVLASVEPUGKQVXOPURMJUWYBLXAFWHPVEUHBVGAGUIZDJULCOZNGEQIC",
//      ...
      "SIZEQJHYGNJDIDFNXMZTJKZXUVUCUIEKGOHRJMRNNDIRIOMTTZSXUMKBXUWA"
    ],
    "signatureHex": "77a9faf00728badab3452b64a9c75e3997128068c4bc3dab1f741264f4f95ece4b11425098debc3984f592aab7d5d827206627a3bdefa24c29797d5065650000"
  }
}
```

***
