## High level description:
The docker-compose deployment system consists of four services:
- `qubic-http` - the http proxy server that provides an http interface to query directly the qubic node(s)
- `qubic-archiver` - the archiver processor and HTTP server that provides rpc endpoints to query the archiver
- `qubic-nodes` - a service responsible with providing `qubic-archiver` and `qubic-http` information regarding reliable nodes and the max tick of the network
- `traefik` - a reverse proxy that will route the incoming requests to the correct service and also exposes metrics and dashboard

# Run with docker-compose
## Prerequisites

Before starting the system, open the `dev.docker-compose.yml` file and make sure you modify it based on your needs:

- Use a reliable peer list for the `qubic-nodes` service. It's defined in the `QUBIC_NODES_QUBIC_PEER_LIST` environment variable. 
You should add your own reliable peers if you have qubic nodes deployed with tx status addon, or you can request a list of reliable peers from the qubic team.
- Use your domain name (or external ip) and replace `testapi.qubic.org` with your domain (or external ip) in the `qubic-http` and `qubic-archiver` services.
- Use your file path and replace volume bindings for the `qubic-archiver` service. For example if you have a store folder on `/root/store/` you should replace volume binding for `qubic-archiver` to `/root/store/archiver:/app/store`.

```bash
$ docker-compose -f dev.docker-compose.yml up -d
```



