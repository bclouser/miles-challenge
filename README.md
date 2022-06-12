
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


## Creating the credentials
The miles-challenge helm chart values.yaml should be updated with all necessary credentials for both strava and google cloud.
### Strava
strava-authorize.txt`
When miles-challenge runs the first time, the logs will display a google-cloud link which must be manually authorized

# To build the miles-challenge app
`cd app`
`docker build . --tag bclouser/miles-challenge:0.0.1`
`docker push bclouser/miles-challenge`
