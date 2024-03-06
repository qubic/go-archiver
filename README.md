## High level description:
The archive system consists of two services:
- `qubic-archiver` - the archiver processor and HTTP server that provides rpc endpoints to query the archiver
- `qubic-node-fetcher` - a service that is starting from a reliable node, gather the peers by "walking" from peer to peer and then filters them out so they don't have more than 30 ticks behind. This service is also exposing an http server for the qubic-archiver to retrieve the reliable peers

## IMPORTANT
Before starting the system, open the `docker-compose.yml` file and make sure that there is a reliable peer as a starting point for the node fetcher. It's defined in the `NODE_FETCHER_QUBIC_STARTING_PEER_IP` environment variable.

## Run with docker-compose:
```bash
$ docker-compose up -d
```

## Available endpoints:

### GetTickTransactions:
```bash
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetTickTransactions -d '{"tickNumber": 12795663}'
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
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetTickData -d '{"tickNumber": 12795663}'
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
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetTransaction -d '{"txId": "qbrdxsmlfggneflkbskpqwjeqhgdssfuabhfapywobqcilroobgvruseuerk"}'
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
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetQuorumTickData  -d '{"tickNumber": 12795663}'

```

### Get Computors
```bash
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetComputors -d '{"epoch": 99}'
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
### GetIdentityInfo
```bash
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetIdentityInfo -d '{"identity": "PJUMDQZEAEDZNFAXATGGYGJULKGALYFCBYUAZRGAHGPXCDBDGEJOMKOFSKKD"}'
{
  "identityInfo": {
    "id": "PJUMDQZEAEDZNFAXATGGYGJULKGALYFCBYUAZRGAHGPXCDBDGEJOMKOFSKKD",
    "tickNumber": 12797233,
    "balance": "1479289941",
    "incomingAmount": "4835212306297",
    "outgoingAmount": "4833733016356",
    "nrIncomingTransfers": 826438,
    "nrOutgoingTransfers": 47187,
    "latestIncomingTransferTick": 12790393,
    "latestOutgoingTransferTick": 12797113,
    "siblingsHex": [
      "2a5af1c66af3ef4a294e09f27aed030d3faeffb9a1910012468b9c7f3e46bd9f",
      "705f57f0c8f11be888bb1e4d935a7582ca65e8212207404bc135b22a6e9bf450",
      "c11fb66d62103d9a2e47b39b205354517d7014e15b6985343daec01f396da1d5",
      "c6b45f22943acd48520886ccd104a580d85b41f39e803c29a8d2eeb0b0e62865"
    ]
  }
}
```

### GetLastProcessedTick
```bash
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetLastProcessedTick
{
  "lastProcessedTick": 12797164,
  "lastProcessedTicksPerEpoch": {
    "99": "12797164"
  }
} 
```



