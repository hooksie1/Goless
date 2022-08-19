# Goless

Goless is a serverless framework for Go functions on Kubernetes.

## Deployment

You can use the Makefile, however if you have a recent version of `kubectl` it will be easier.

### CRD setup

`kubectl kustomize config/crd | kubectl apply -f -`

### Controller Deployment

`kubectl kustomize config/default | kubectl apply -f -`

### Function Deployment

A sample function is in `config/samples/sample.yaml`


A Goless function looks like this:

```
apiVersion: goless.io/v1beta1
kind: Function
metadata:
  name: function-example
spec:
  service: "example"
  serverPort: 9000
  replicas: 1
  function: |
    package handlers
    import (
      "net/http"
    )
    func Handler(w http.ResponseWriter, r *http.Request) {
      w.Write([]byte("Hey this example works!"))
    }
```

Since this is the default Go HTTP package we can include middleware as well:

```
apiVersion: goless.io/v1beta1
kind: Function
metadata:
  name: function-example
spec:
  service: "example"
  serverPort: 9000
  replicas: 1
  function: |
    package handlers
    import (
      "net/http"
    )

    func Handler(w http.ResponseWriter, r *http.Request) {
      w.Header().Add("foo", "bar")

      Foo().ServeHTTP(w, r)

    }

    func Foo() http.HandlerFunc {
      return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("in handler"))
      })
    }
```

## Removal

### CRDs

`kubectl kustomize config/crd | kubectl delete -f -`

### Operator

`kubectl kustomize config/default | kubectl delete -f -`