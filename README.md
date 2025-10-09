# README

## Running a server

./s3-baby-server serve ~/pool-s3bbs --addr 127.0.0.1:9000 --auth-key s3baby,s3baby

## Restrictions

- File names cannot begin with a dot ("."), they are hidden.

## Additional Features

- s3bbs stores access logs if a ".access-log" directory exists.

----------------------------------------------------------------

# サーバーの起動
**必須引数**
| コマンド | デフォルト値 | 説明 |
| --- | --- | --- |
| serve PATH | None | Specify the directory to serve in PATH |

```
go run . serve ../s3_baby_server_test
```

**オプション引数**  
| コマンド | デフォルト値 | 説明 |
| --- | --- | --- |
| addr | 127.0.0.1:9000 | IPaddress:Port |
| logPath | None | Log output path |
| auth-key | admin,admin | Set key pair: access_key_id, secret_access_key |

認証キーに関しては環境変数**AUTH_KEY**を変数、"access key","secret access key"を値に指定することで設定可能

```
go run . serve ../s3_baby_server_test --logPath serverLog.log
```

# API
- AbortMultipartUpload
- CompleteMultipartUpload
- CopyObject
- CreateBucket
- CreateMultipartUpload
- DeleteBucket
- DeleteObject
- DeleteObjects
- DeleteObjectTagging
- GetObject
- GetObjectAttributes
- GetObjectTagging
- HeadBucket
- HeadObject
- ListBuckets
- ListMultipartUploads
- ListObjects
- ListObjectsV2
- ListParts
- PutObject
- PutObjectTagging
- UploadPart
- UploadPartCopy


# ディレクトリ構成
```
/s3-baby-server
├── cmd
│    └── server_cmd_analyze.go(引数確認)
│
├── internal
│    ├── api（各APIごとの受け口）
│    │    ├── handler_base.go（各APIの共通処理をまとめるベースハンドラ）
│    │    ├── http3options.go（HTTPリクエストに関する構造体）
│    │    ├── abort_multipart_upload.go
│    │    ├── complete_multipart_upload.go
│    │    ├── copy_object.go
│    │    │
│    │
│    │
│    ├── model（データ構造）
│    │    ├── s3option_model.go（データの受け渡し用の構造体）
│    │    ├── s3response_model.go（レスポンス形式にまとめる構造体）
│    │    ├── set_checksum.go（チェックサムの結果をレスポンス形式にまとめる構造体）
│    │    ├── complete_multipart_upload.go（リクエストの値、処理結果をまとめる構造体）
│    │    ├── copy_object.go
│    │
│    │
│    ├── server（初期設定）
│    │    ├── recover.go（panicリカバリ）
│    │    ├── server.go（サーバーの初期化）
│    │    └── logger.go（ログ作成）
│    │
│    │
│    └── service（各APIで使う機能）
│         ├── filesystem.go（ファイルシステムを扱う構造体）
│         ├── s3multipart.go（マルチパートアップロードを扱う構造体）
│         ├── s3tag_utils.go（タグを扱う構造体）
│         ├── s3processing.go（各APIで指定されるパラメータの内部処理を扱う構造体）
│         ├── s3error.go（S3特有のエラー構造体）
│         ├── s3options.go（http3optionsのインターフェース）
│         └── s3service.go（各APIのリクエスト受け取り、結果を返す）
│
│
├── pkg
│    └── utils（共通処理を入れる）
│         ├── check_size_utils.go（上限値のチェック処理）
│         ├── convert_utils.go（データ型変換処理）
│         ├── file_utils.go（ファイル操作処理）
│         └── hash_utils.go（ハッシュ計算処理）
│
│
└── main.go(server_cmd_analyze.goの呼び出し)
```