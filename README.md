# docker run -v /root/store:/app/store --name qubic-archiver --restart=always -d ghcr.io/qubic/qubic-archiver

## Docker usage:
When running the docker container, you need to specify the following environment variables:
- `QUBIC_ARCHIVER_QUBIC_NODE_IP` - the IP address of the Qubic node that archiver will connect to
- `QUBIC_ARCHIVER_QUBIC_FALLBACK_TICK` - the start tick for the archiver to start archiving from (needs to be from current epoch)
```bash
$ docker run -p 8000:8000 -p 8001:8001 -e QUBIC_ARCHIVER_QUBIC_NODE_IP="212.51.150.253" -e QUBIC_ARCHIVER_QUBIC_FALLBACK_TICK=12543674 -v /root/store:/app/store --name qubic-archiver -d ghcr.io/qubic/qubic-archiver
```

## Querying the archiver
```bash
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetTickTransactions -d '{"tickNumber": 12570631}'
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetTickData -d '{"tickNumber": 12570631}'
$ curl -sX POST  http://127.0.0.1:8000/qubic.archiver.archive.pb.ArchiveService/GetTransaction -d '{"txId": "qbizhhehgiijddcgbzdxqtkqcwuedcvgnrorvovdkavytqsukxlucyjhjxeg"}'
```