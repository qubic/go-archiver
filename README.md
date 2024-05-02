## High level description:

The archive system consists of two services:

- `qubic-archiver` - the archiver processor and HTTP server that provides rpc endpoints to query the archiver
- `qubic-nodes` - a service responsible with providing information regarding reliable nodes and the max tick of the
  network

## IMPORTANT

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
```

## Run with docker-compose:

```bash
$ docker-compose up -d
```

## Configuration

## Available endpoints:

### GetTickTransactions:

```bash
$ curl http://127.0.0.1:8001/ticks/{tick}/transactions
{
  "transactions": [
    {
      "sourceId": "PJUMDQZEAEDZNFAXATGGYGJULKGALYFCBYUAZRGAHGPXCDBDGEJOMKOFSKKD",
      "destId": "AFZPUAIYVPNUYGJRQVLUKOPPVLHAZQTGLYAAUUNBXFTVTAMSBKQBLEIEPCVJ",
      "amount": "0",
      "tickNumber": 12795663,
      "inputType": 0,
      "inputSize": 32,
      "inputHex": "716c692d637564614dfd03e48d610cd450b590e835155b6b9f55f91c516428b4",
      "signatureHex": "1e597cb8f834797a2b58e8178f7f18219db4797e803277632299ebdc222a4e5194037c29d8dd87c2f8cebdf8dfb30f34b175e37565f8b01d059dc71cc1342200",
      "txId": "qbrdxsmlfggneflkbskpqwjeqhgdssfuabhfapywobqcilroobgvruseuerk"
    }
  ]
}
```

### GetTickData

```bash
$ curl http://127.0.0.1:8001/ticks/{tick}/tick-data
{
  "tickData": {
    "computorIndex": 335,
    "epoch": 99,
    "tickNumber": 12795663,
    "timestamp": "1709731981000",
    "varStruct": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
    "timeLock": "Nd5Fn0wHnQmcar+qphSK5ucbEtg/gkY82ffWs4HBbkk=",
    "transactionIds": [
      "qbrdxsmlfggneflkbskpqwjeqhgdssfuabhfapywobqcilroobgvruseuerk"
    ],
    "contractFees": [],
    "signatureHex": "4aef8b84c2b594331eae649722ed747fa01a69b37e38538a0e3257c8f739950c7c7433d22d6fe77231972d51927fb29b3b4cc6dc9b202e609d42e5eddafe1c00"
  }
}
```

### GetTransaction

```bash
$ curl http://127.0.0.1:8001/transactions/{tx_id}
{
  "transaction": {
    "sourceId": "PJUMDQZEAEDZNFAXATGGYGJULKGALYFCBYUAZRGAHGPXCDBDGEJOMKOFSKKD",
    "destId": "AFZPUAIYVPNUYGJRQVLUKOPPVLHAZQTGLYAAUUNBXFTVTAMSBKQBLEIEPCVJ",
    "amount": "0",
    "tickNumber": 12795663,
    "inputType": 0,
    "inputSize": 32,
    "inputHex": "716c692d637564614dfd03e48d610cd450b590e835155b6b9f55f91c516428b4",
    "signatureHex": "1e597cb8f834797a2b58e8178f7f18219db4797e803277632299ebdc222a4e5194037c29d8dd87c2f8cebdf8dfb30f34b175e37565f8b01d059dc71cc1342200",
    "txId": "qbrdxsmlfggneflkbskpqwjeqhgdssfuabhfapywobqcilroobgvruseuerk"
  }
}
```

### GetQuorumTickData

```bash
$ curl http://127.0.0.1:8001/ticks/{tick_number}/quorum-tick-data
```

### Get Computors

```bash
$ curl http://127.0.0.1:8001/epochs/{epoch}/computors
{
  "computors": {
    "epoch": 99,
    "identities": [
      "9263520905f41451c102f3e4c9a2434490cd766ceaac5f03a5300f7b5a5e60c7",
      "b23fc58a4371aedac687ef76c2ebc666ac554f1aee07351a9aa503b6df05ad92",
      "0d6fa2d10c02f7cc33fec302bd77535d91e9c905785c6b0e7f03ef7547279174",
      "e245860121add2513340b0c47df7010410120a41368dcdb5246b581f2a63d99f",
      "27402616c685f579e233e64056ad5553fd475541439e68ff6587da0f9ee05e8b",
      "8931ddf58887c72eb693ab866212b620d7db79ba1ec1456826d67e313c7e50c5"
    ],
    "signatureHex": "77d93167d5cf9ba7aca8139fcbdb0624eaf6884d7a3bb79832dc5f59131f84da55ee9e29f81cd07fae746b4b7aa055b734c58ce21c3443aef5da521903d10d00"
  }
}
```

### GetStatus

```bash
$ curl http://127.0.0.1:8001/status
{
   "lastProcessedTick":{
      "tickNumber":13682425,
      "epoch":107
   },
   "lastProcessedTicksPerEpoch":{
      "106":13548510,
      "107":13682425
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
               "lastProcessedTick":13682425
            }
         ]
      }
   ]
}
```
