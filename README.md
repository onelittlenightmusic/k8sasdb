# K8sAsDB (k8sasdb)

Use Kubernetes as a database.

k8sasdb creates a new resource (`Table`) according to your request.

## Usage

### 1. Create a table

Create this file as `file-create-table.yaml` and run the next `kubectl` command.
```yaml
apiVersion: db.k8sasdb.org/v1
kind: Table
metadata:
  name: fruit
spec:
  columns:
  - name: sweetness
    type: bool
  - name: weight
    type: int
  - name: comment
    type: string
```

```sh
kubectl apply -f file-create-table.yaml
```

### 2. Create a record in the table

Create this file as `file-create-record.yaml` and run the next `kubectl` command.
```yaml
apiVersion: user.k8sasdb.org/v1
kind: Fruit
metadata:
  name: orange
spec:
  test: success
```

```sh
kubectl apply -f file-create-record.yaml
```

### 3.1 List records in a table

```sh
kubectl get fruits
# output
NAME     AGE
apple    9s
orange   9s
```

### 3.2 Get a record

```sh
kubectl get fruits orange -o yaml
# output
apiVersion: user.k8sasdb.org/v1
kind: Fruit
metadata:
  annotations:
    ...
spec:
  test: success
```

### 4. Delete a record

```sh
kubectl delete fruit orange
```

## Installation

```sh
kubectl apply -f https://raw.githubusercontent.com/onelittlenightmusic/k8sasdb/master/install.yaml
```

## Try sample request

```sh
kubectl apply -f test/fruit.yaml

kubectl apply -f test/apple.yaml
kubectl apply -f test/banana.yaml
```

## Tear down 

```sh
kubectl delete -f https://raw.githubusercontent.com/onelittlenightmusic/k8sasdb/master/install.yaml
```