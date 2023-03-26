Golang demo app
===============================



Dockerが認証にGoogle Cloud SDKを使用する設定
--------------------

```sh
gcloud auth configure-docker asia-docker.pkg.dev
```

Build & Run
-------------------------

```
$ skaffold build
$ skaffold run
$ skaffold dev --cleanup=false
```

or Cloud Build

```sh
gcloud builds submit -t asia-docker.pkg.dev/dtamura-service01/containers/golang-demo-ping:latest
```


Require
--------------------
- docker
- gcloud
- skaffold
