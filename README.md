# 体重管理自動化アプリ

- Excelを使ったスポーツチームの体重管理フローの自動化プロジェクト
- 詳細は[こちら](https://zenn.dev/hsaitooo/articles/50c0e66f9820cb)の記事を参照

## 旧業務フロー
![旧業務フロー](./doc/old_flow.svg)

## 構成
![構成](./doc/architecture.svg)

``` bash
sam deploy --guided
aws s3 cp index.html s3://{hostingBucket}
aws s3 cp {resource.csv} s3://{resouceBucket}
```
