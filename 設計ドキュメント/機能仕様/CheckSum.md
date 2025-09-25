# チェックサム

## 概要
データをアップロード・ダウンロード・コピーする際に、データの整合性を検証するための機能。

## API
- CopyObject
- GetObject
- HeadObject
- PutObject
- PutObjectTagging
- DeleteObjects
- CreateMultipartUpload
- CompleteMultipartUpload
- UploadPart
- UploadPartCopy
- GetObjectAttributes
- ListParts

## アルゴリズム一覧
- CRC-64/NVME (CRC64NVME)
- CRC-32 (CRC32)
- CRC-32C (CRC32C)
- SHA-1 (SHA1)
- SHA-256 (SHA256)

## 使用パッケージ
- github.com/aws/aws-sdk-go-v2

## 計算方法
1. チェックサム関数を生成(`使用パッケージ参照`)
2. ファイル内容をバイト列に変換
3. チェックサム関数でバイト列を16進数に変換
4. 16進数の値をNバイトの値に書き出し（アルゴリズムによってNバイト数は異なる）
5. 出力されたNバイトの値を(`encoding/base64`)で文字列に変換

## マルチパートアップロード時の計算方法
マルチパートアップロードのチェックサム計算は`CreateMultiPartUpload`のオプションで以下のタイプに指定が可能。
- フルオブジェクトチェックサム
    1. アップロード前にチェックサム値を算出
    2. パートファイルを順次アップロード
    3. アップロード完了後、結合したファイルからチェックサム値を算出
    4. アップロード前後のチェックサム値を比較
    5. OKならアップロード先に保存

- 複合チェックサム
    1. アップロード前にチェックサム値を算出
    2. パートファイル毎にチェックサム値を算出
    3. サーバーに全て送信後、パートファイルを結合し、チェックサム値も合算
    4. 合算したチェックサム値とアップロード前に算出したチェックサム値を比較
    5. OKならアップロード先に保存

