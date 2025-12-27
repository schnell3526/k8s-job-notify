# k8s-job-notify

Kubernetes Jobの完了/失敗をSlackに通知するコントローラー

## ローカルでの動作確認（kind）

> **Note**: kindはDevContainerの外（ホストマシン）で実行してください。
> DevContainerはdocker-outside-of-dockerを使用しており、kind内部のkubectl実行時にAPIサーバーへの接続問題が発生します。

### 1. クラスタ作成

```bash
kind create cluster --config ./k8s/kind-cluster.yaml
```

### 2. イメージのビルドとロード

```bash
# イメージをビルド
docker build -t k8s-job-notify:latest -f docker/Dockerfile .

# kindクラスタにイメージをロード
kind load docker-image k8s-job-notify:latest
```

### 3. デプロイ

```bash
# Slack Webhook URLを設定してからデプロイ
# k8s/deployment.yaml の webhook-url を編集するか、以下のようにSecretを別途作成
kubectl create namespace k8s-job-notify
kubectl create secret generic slack-webhook \
  -n k8s-job-notify \
  --from-literal=webhook-url="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

# デプロイ（Secretを別途作成した場合はdeployment.yamlからSecret部分を削除）
kubectl apply -f k8s/deployment.yaml
```

### 4. 動作確認

```bash
# コントローラーのログを確認
kubectl logs -f -n k8s-job-notify deployment/k8s-job-notify

# テスト用Jobを実行（別ターミナル）
kubectl apply -f k8s/test-job.yaml

# Job状態を確認
kubectl get jobs -n default
```

### 5. クリーンアップ

```bash
# テストJobの削除
kubectl delete -f k8s/test-job.yaml

# クラスタ削除
kind delete cluster
```

## 本番環境へのデプロイ

### イメージのプッシュ

```bash
# 例: GitHub Container Registry（マルチプラットフォーム）
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/<owner>/k8s-job-notify:v1.0.0 \
  -f docker/Dockerfile \
  --push .
```

### マニフェストの更新

`k8s/deployment.yaml`のイメージを変更：

```yaml
image: ghcr.io/<owner>/k8s-job-notify:v1.0.0
imagePullPolicy: Always
```

プライベートレジストリの場合は`imagePullSecrets`を追加してください。
