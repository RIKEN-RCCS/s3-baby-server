# サーバー管理全体構造設計書
## 1. 概要
ディレクトリ構成、メタデータ、パラメータ解析について記載。


## 2. ディレクトリ構成
```
/root/
   ├── バケット名1/
   │     ├── ファイル1
   │     ├── .ファイル1_meta.json
   │     ├── ファイル2
   │     └── .ファイル2_meta.json
   ├── バケット名2/
   │     ├── ファイル1
   │     └── .ファイル1_meta.json
   │
   └── .S3BabyServer
         ├── MultiPartUpload/
         │     ├── UploadID1/ ←CompleteMultipartUploadで結合確認出来たら削除
         │     │     ├── UploadID1_meta.json（内容：一時ファイル）
         │     │     ├── PartNumber1（一時ファイル）←AbortMultipartUploadを使用したときにも削除
         │     │     └── PartNumber2（一時ファイル）
         │     └── UploadID2/
         │           ├── UploadID2_meta.json（内容：一時ファイル）
         │           ├── PartNumber1（一時ファイル）
         │           └── PartNumber2（一時ファイル）
         │
         └── TmpUpload/
               └── アップロード中の一時ファイル
```


## 3. 管理設計
### バケット、オブジェクト
- ユーザーが指定したディレクトリがルートディレクトリとなる。
- バケット名としては`A-Z, a-z, 0-9`を開始文字とし、`A-Z, a-z, 0-9, -`を終了文字とする。リクエスト確認時に開始文字、終了文字として使用できない文字であればエラーを返す。
- ETagに関しては[**ETag.md**](../機能仕様/ETag.md)ファイルを参照。
- `/root/.S3BabyServer/`, `/root/.S3BabyServer/MultiPartUpload`, `/root/.S3BabyServer/TmpUpload`ディレクトリは隠しディレクトリとする。
- オブジェクトの指定ではルートディレクトリから見た相対パスで判断する。

### UploadID作成
- 自然数をUploadIDとする。
- 新しくUploadIDが作成される場合はMultiPartUploadディレクトリ内を確認し、自然数の最大値を確認し、インクリメントしたUploadIDを使用する。
```
例）MultiPartUploadディレクトリ内が空の場合→1をUploadIDとしディレクトリを作成する。  
　　MultiPartUploadディレクトリ内に4までのUploadIDディレクトリが存在する場合→5をUploadIDとしディレクトリを作成する。
```

### ファイル書き込み
- 書き込みは一時ファイルとして行い、失敗時には削除する。
- 一時ファイルの格納先としては/root/.S3BabyServer/TmpUploadとし、成功時に指定のパスに変更する。
- `/root/.S3BabyServer/TmpUpload`ディレクトリに中断された一時ファイルが複数存在する場合、ストレージを圧迫する恐れがあるためサーバー起動時にディレクトリ内を空にする。
- 同じパスが指定された場合は**元のオブジェクトを上書きする。**
- 一時ファイルの名前として、オブジェクト名をMD5関数に入力し出力された値を使用する。
```
例）test_object.txtをアップロード
1. 拡張子を抜いた文字列（test_object）をMD5関数に入力
2. 得られた値（509813693a3477df0440e10ca358539c）+.tmpを一時ファイル名に使用
   `509813693a3477df0440e10ca358539c.tmp`
3. 成功時に509813693a3477df0440e10ca358539c.tmp → test_object.txtにファイル名を変更
```

### メタデータ
- Tagを付与するときにメタデータファイルを作成する。
- Tagを指定して操作をする場合が考えられるためメタデータとして保持をする。
- マルチパートアップロードは[**MultipartUpload.md**](../機能仕様/MultipartUpload.md)に記載。
- ETag、LastModifiedなどはメタデータとして保持せず、オプションで指定があった場合にファイルシステムを使用し取得する。
- タグの変更、削除があった場合にメタデータも変更、削除し、ファイル削除の際はメタデータも削除する。
- 新規でオブジェクトが作成されたときにメタデータに必要な情報を算出し、該当のディレクトリに格納する。
```
例1）ファイル1のメタデータ（/root/バケット名1/ファイル1_meta.json）
{
  "Tag": [
    {"Key": 1, "Value": "tag1"},
    {"Key": 2, "Value": "tag2"}
  ]
}
```

### ログ機能
- コンソール出力はデフォルトで行う。
- 別途、引数で指定された場合はファイルにログの出力を行う。
- 既にログファイルが存在する場合内容を追記する。
- ローリング機能は実装しない。
- ログ出力を行うタイミングとしては以下のタイミングに行う。
  - サーバー起動時
  - APIリクエスト/レスポンス時（どのAPIが呼び出されたか、ステータスコードはなにで終了したか）
  - エラー時


## 4. パラメータ解析
リクエストされたAPIから必要なパラメータ、エラーを返すパラメータを解析し処理。  
処理が完了次第レスポンスをクライアントに返す。

### クライアントからAPIサーバー
#### APIサーバー
- HTTPリクエストメソッド：
- ヘッダー：
- ボディ：

-内容例：
```
PUT /my-second-image.jpg HTTP/1.1
Host: amzn-s3-demo-bucket.s3.<Region>.amazonaws.com
Date: Wed, 28 Oct 2009 22:32:00 GMT
x-amz-copy-source: /amzn-s3-demo-bucket/my-image.jpg
```

### APIサーバーからクライアント
- HTTPステータスコード：
- ヘッダー：
- ボディ：

-内容例：
```
HTTP/1.1 200 OK
Server: AmazonS3
Date: Wed, 28 Oct 2009 22:32:00 GMT
x-amz-id-2: eftixk72aD6Ap51TnqcoF8eFidJG9Z/2mkiDFu8yU9AS1ed4OpIszj7UDNEHGran
x-amz-request-id: 318BC8BC148832E5
<CopyObjectResult>
  <LastModified>2009-10-12T17:50:30.000Z</LastModified>
  <ETag>"9b2cf535f27731c974343645a3985328"</ETag>
</CopyObjectResult>
```
