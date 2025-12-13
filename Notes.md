# How to Generate Self-signed CA and cert pairs for m-TLS ?

We have some example cfssl JSON files in `certs/manifests`, you can generate example
self-signed CA cert pair and peer cert pairs with commands:

```shell
cd certs
./gen-ca.pem
./gen-cert-pair.sh manifests/hub.json
./gen-cert-pair.sh manifests/agent1.json
```

If you haven't installed cfssl executables, install them first:

```shell
cd cfssl
make
make install
```
