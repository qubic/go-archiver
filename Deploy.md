## High level description:
The docker-compose deployment system consists of four services:
- `qubic-http` - the http proxy server that provides an http interface to query directly the qubic node(s)
- `qubic-archiver` - the archiver processor and HTTP server that provides rpc endpoints to query the archiver
- `qubic-node-fetcher` - a service that is starting from a reliable node, gather the peers by "walking" from peer to peer(if not set to fixed list) and then filters them out so they don't have more than 30 ticks behind. This service is also exposing an http server for the qubic-archiver to retrieve the reliable peers
- `traefik` - a reverse proxy that will route the incoming requests to the correct service and also exposes metrics and dashboard

# Run with docker-compose
## Prerequisites

Before starting the system, open the `dev.docker-compose.yml` file and make sure you modify it based on your needs:

- Use reliable peer list for the node-fetcher service. It's defined in the `NODE_FETCHER_QUBIC_STARTING_PEERS_IP` environment variable. 
You should add your own reliable peers if you have qubic nodes deployed with tx status addon, or you can request a list of reliable peers from the qubic team.
- Use your domain name (or external ip) and replace `testapi.qubic.org` with your domain (or external ip) in the `qubic-http` and `qubic-archiver` services.
- Use your file path and replace volume bindings for `qubic-archiver` and `qubic-node-fetcher` services. For example if you have a store folder on `/root/store/` you should replace volume binding for `qubic-archiver` to `/root/store/archiver:/app/store`.

```bash
$ docker-compose -f dev.docker-compose.yml up -d
```



