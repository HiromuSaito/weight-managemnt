# 体重管理自動化アプリ

詳細は[こちら](https://zenn.dev/hsaitooo/articles/50c0e66f9820cb)

## 置き換え前フロー
![](./architecture.drawio.svg)

## AWSでのフロー

![](./old_flow.drawio.svg)

``` bash
sam deploy --guided
aws s3 cp index.html s3://{hostingBucket}
aws s3 cp {resource.csv} s3://{resouceBucket}
```