# K8sAsDB (k8sasdb)

Use Kubernetes as a database.

k8sasdb creates a new resource (`Table`) according to your request.

## TL;DR

### 1. Create a table

```yaml
apiVersion: db.k8sasdb.org/v1
kind: Table
metadata:
  name: fruit
spec:
  columns:
  - name: test
    type: string
```

### 2. Create a record in the table

```yaml
apiVersion: user.k8sasdb.org/v1
kind: Fruit
metadata:
  name: orange
spec:
  test: success
```

```sh
kubectl apply -f <file>.yaml
```

### 3. List records in a table

```sh
kubectl get fruits
```

### 4. Delete a record

```sh
kubectl delete fruit orange
```

## Installation

```sh
make install
make run
```

## Try sample request

```sh
kubectl apply -f config/samples/
```

## Tear down 

```sh
make uninstall
```