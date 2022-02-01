
# To deploy

## Traefik
We use traefik to handle the tls stuffs
```
helm upgrade --install traefik traefik/traefik --version 9.1.1 -f ./traefik-values.yaml
```

## Miles Challenge
Helm chart in deploy/app

Helm chart requires a pvc to hold data such as credentials and whatnot.

It pulls a docker image from dockerhub which can be built by following the build directions below

```
helm upgrade --install miles-challenge ./deploy/miles-challenge/
```




# To build the miles-challenge app
`cd app`
`docker build . --tag bclouser/miles-challenge:0.0.1`
`docker push bclouser/miles-challenge`
