# マルチパートアップロード
## 一時保存構成
パートナンバーファイルを変更、削除ができるように完了するまではデータを保持
1. UploadIDディレクトリを作成
2. ルートパス: `/root/.S3BabyServer/MultiPartUpload/UploadId`
3. 各パートを連番ファイルとして保存（例: `1`, `2`, `3`, ...）
4. 保存が完了したパートの内容を`UploadID1_meta.json`に保存
- `UploadID1_meta.json`の内容例:
```json
{
  "Bucket": "バケット名",
  "Key": "ファイル名",
  "Parts": [
    {"PartNumber": 1, "ETag": "etag1"},
    {"PartNumber": 2, "ETag": "etag2"}
  ]
}
```

## アップロード完了時
CompleteMultipartUploadが複数リクエストされた場合、古いリクエストは無効にし、新しいリクエストを有効とする。
1. CompleteMultipartUploadで指定されたパートを順番に読み込み結合。
2. 結合したファイルを本ファイルとして保存。
3. 本ファイル名_meta.jsonにMultipartETagをキー、バリューを結合後のETagとした記載を追加。
4. `/root/.S3BabyServer/MultiPartUpload/UploadId`を削除。


## ローカル→サーバー
## 概要
大規模ファイルをローカルからサーバーに上げる場合の処理  
*クライアントはファイル容量によって（閾値）マルチパートアップロードへの切り替えを行う*

## API
- CreateMultipartUpload
- UploadPart
- CompleteMultipartUpload

## シーケンス図
```mermaid
sequenceDiagram
    autonumber
    participant クライアント
    participant サーバー
    participant ファイルシステム

    %% 正常動作
    クライアント->>サーバー: CreateMultipartUpload [Bucket, Key]
    サーバー->>ファイルシステム: UploadIDディレクトリ作成
    サーバー-->>クライアント: UploadId

    loop パートファイル分繰り返す
        クライアント->>サーバー: UploadPart [PartNumber, UploadId]
        サーバー->>ファイルシステム: パートファイル保存
        サーバー-->>クライアント: ETag
    end

    %% 分割ファイルの合体
    クライアント->>サーバー: CompleteMultipartUpload [Part, UploadId, ETag]
    サーバー->>ファイルシステム: 結合ファイル保存
    サーバー-->>クライアント: 200 OK

```

## ディレクトリ→ディレクトリ
### 概要
大規模ファイルをサーバー間でコピーする際の処理  
*閾値によってマルチパートアップロードへの切り替えを行う*

### API
- HeadObject
- GetObjectTagging
- CreateMultipartUpload
- UploadPartCopy
- CompleteMultipartUpload


### シーケンス図
```mermaid
sequenceDiagram
    autonumber
    participant クライアント
    participant サーバー
    participant ファイルシステム


    %% 正常動作
    クライアント->>サーバー: HeadObject [Bucket, Key]
    サーバー->>ファイルシステム: オブジェクトのメタデータ取得
    サーバー-->>クライアント: 200 OK

    クライアント->>サーバー: GetObjectTagging
    サーバー->>ファイルシステム: タグ取得
    サーバー-->>クライアント: Tagging

    クライアント->>サーバー: CreateMultipartUpload [Bucket, Key]
    サーバー->>ファイルシステム: UploadIDディレクトリ作成
    ファイルシステム-->>クライアント: UploadId

    loop パートファイル分繰り返す
        クライアント->>サーバー: UploadPartCopy [PartNumber, UploadId]
        サーバー->>ファイルシステム: パートファイル保存
        サーバー-->>クライアント: ETag
    end

    %% 分割ファイルの合体
    クライアント->>サーバー: CompleteMultipartUpload [Part, UploadId、ETag]
    サーバー->>ファイルシステム: 結合ファイル保存
    サーバー-->>クライアント: 200 OK

```