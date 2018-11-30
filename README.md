# TrafficClaim Enforcer

The TrafficClaim Enforcer implements the intent of the TrafficClaim resource
described in [RFE: Traffic
Ownership](https://docs.google.com/document/d/1QD82w2mOPbQRaPDiesDNingDFfH_bZ4K9E5BALrAtiI/view)
for a Kubernetes cluster with the Istio service mesh.

A TrafficClaim is a Kubernetes config resource that says which network traffic
can be controlled by Kubernetes objects (like VirtualService) in a namespace.

## Intro

A single Istio mesh is designed to manage traffic for many services often owned
by different teams. 

In order to understand and configure traffic, service owners should be
confident that the intended config will be applied to traffic arriving at their
services by trusting the rules that allow the config affecting that traffic
from being created in the first place.

## Background

Istio 1.0’s traffic management config resources allow traffic for any service
to be altered, routed to different services, etc.

This makes the actual traffic config that Istio is implementing unpredictable
without reviewing all Istio config that exists. For example, any other
VirtualService resource could refer to a hostname, which may conflict or
override the intended config, causing the actual config to be different. This
may be accidental or malicious, and affects several resource types.

To guarantee that only the intended config is applied, additional restrictions
on what is considered valid config is needed, so that the config can be
specified along service ownership boundaries.

## TrafficClaim

A TrafficClaim is a Kubernetes resource that defines the service ownership
boundaries.  Here's an example:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: TrafficClaim
metadata:
  name: hostname-claim-example
  namespace: team-a
spec:
  claims:
  - hosts:
    - *.foo.com
    - myteam.bar.com
    ports:
    - 80
    - 8080
    http:
      paths:
      - prefix: /svcA
      - exact: /some/exact/path
```

## TrafficClaim Enforcer

The TrafficClaim enforcer runs as a Kubernetes ValidationWebhook.  Whenever a
user attempts to configure a new VirtualService, it checks if the
VirtualService is only affecting hosts, ports, or paths that are valid
according to the TrafficClaims.  If so, it allows the VirtualServer; if not it
returns an error message to the user.

## Installing

Kubernetes webhooks need to authenticate to the Kubernetes API server.  There's
a script in install/create-webhook-certs.sh (based on a script from Istio) to
generate the right certs by submitting a CSR to kubeapi.  Next, you need to
deploy the TCE to the cluster, and then create a Webhook resource to tell
Kubernetes to call it.

1. Make sure your kubeconfig is pointed at the right cluster (`kubectl config
get-contexts` or similar)

2. `kubectl apply -f install/tce.yaml`

This step installs the Deployment, the Service and the webhook configuration so
that kubernetes can consult our trafficclaim.  It also starts a job to get a
signed certificate so that kubernetes will trust our webhook when it calls us.
The enforcer won't work until that job completes, which usually only takes a
few seconds.

## Using

Try creating a new VirtualServer without a TrafficClaim:

```
kubectl apply -f install/test.yaml
<error message>
```

Now install a TrafficClaim:

```
kubectl apply -f install/test-trafficclaim.yaml
kubectl apply -f install/test.yaml
<success message>
```

You can try it with your own resources and traffic claims too.

In this configuration, the TrafficClaim is only enforced in namespaces with the
"tce: enabled" annotation.  This helps while you are experimenting - if the TCE
is uavailable temporarily you can still create policy.  Add the "tce: enabled"
annotation to namespaces:

```
kubectl annotate ns default tce=enabled
```

## Building

`make` to compile everything.  `make images` to produce a docker image.
