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

# How to Prepare the Development Environment ?

1. Edit the hosts file (/etc/hosts) to make agent1.example.one and hub.example.com both points to 127.0.0.1 and ::1.
2. Generate self-signed Root CA, server cert pairs, and client cert pairs as stated above.
3. Invoke handy scripts in scripts/ folder, to launch hub, then the agent.
4. Hub listens :8080 for accepting agents registration request, :8082 for serving public un-authenticated requests.
5. Agent listens :8081 for serving requests, and the transport is an m-TLS protected, however the cert pairs generated above can both use at server-auth and client-auth, so it's fine.
