// handlers.go (2025-11-07)
// API-STUB.  Handler functions (h_XXXX) called from the
// dispatcher.
package server
import (
"context"
"encoding/xml"
"fmt"
"io"
"log"
"net/http"
"slices"
"strconv"
"strings"
"time"
"github.com/aws/aws-sdk-go-v2/service/s3"
"github.com/aws/aws-sdk-go-v2/service/s3/types"
)
// BB_ENUM_INTERN_ERROR is an error returned when interning
// an enumeration.
type Bb_enum_intern_error struct {
Enum string
Name string
}
func (e *Bb_enum_intern_error) Error() string {
return "Enum " + e.Enum + " unknown key: " + strconv.Quote(e.Name)}
// BB_INPUT_ERROR is recorded in a context when an error
// occurs on interning a parameter.
type Bb_input_error struct {
Key string
Err error
}
func (e *Bb_input_error) Error() string {
return "Parameter " + e.Key + " error: " + e.Err.Error()}
func h_AbortMultipartUpload(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.AbortMultipartUploadInput{}
if len(hi.Values("x-amz-if-match-initiated-time")) != 0 {
var s = hi.Get("x-amz-if-match-initiated-time")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-if-match-initiated-time"] = &Bb_input_error{"x-amz-if-match-initiated-time", err2}} else {i.IfMatchInitiatedTime = &x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if qi.Has("uploadId") {
i.UploadId = thing_pointer(qi.Get("uploadId"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.AbortMultipartUpload(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_AbortMultipartUploadResponse(*o)
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 204
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_CompleteMultipartUpload(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.CompleteMultipartUploadInput{}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if len(hi.Values("If-None-Match")) != 0 {
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))}
if len(hi.Values("If-Match")) != 0 {
i.IfMatch = thing_pointer(hi.Get("If-Match"))}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-mp-object-size")) != 0 {
var s = hi.Get("x-amz-mp-object-size")
var x, err2 = strconv.ParseInt(s, 10, 64)
if err2 != nil {input_errors["x-amz-mp-object-size"] = &Bb_input_error{"x-amz-mp-object-size", err2}} else {i.MpuObjectSize = &x}}
if len(hi.Values("x-amz-checksum-type")) != 0 {
var s = hi.Get("x-amz-checksum-type")
var x, err2 = intern_ChecksumType(s)
if err2 != nil {input_errors["x-amz-checksum-type"] = &Bb_input_error{"x-amz-checksum-type", err2}} else {i.ChecksumType = x}}
if len(hi.Values("x-amz-checksum-sha256")) != 0 {
i.ChecksumSHA256 = thing_pointer(hi.Get("x-amz-checksum-sha256"))}
if len(hi.Values("x-amz-checksum-sha1")) != 0 {
i.ChecksumSHA1 = thing_pointer(hi.Get("x-amz-checksum-sha1"))}
if len(hi.Values("x-amz-checksum-crc64nvme")) != 0 {
i.ChecksumCRC64NVME = thing_pointer(hi.Get("x-amz-checksum-crc64nvme"))}
if len(hi.Values("x-amz-checksum-crc32c")) != 0 {
i.ChecksumCRC32C = thing_pointer(hi.Get("x-amz-checksum-crc32c"))}
if len(hi.Values("x-amz-checksum-crc32")) != 0 {
i.ChecksumCRC32 = thing_pointer(hi.Get("x-amz-checksum-crc32"))}
if qi.Has("uploadId") {
i.UploadId = thing_pointer(qi.Get("uploadId"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
{var x types.CompletedMultipartUpload
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {panic(fmt.Errorf("No http body for types.CompletedMultipartUpload: %w", err1))}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {panic(fmt.Errorf("Invalid http body for types.CompletedMultipartUpload: %w", err2))}
i.MultipartUpload = &x}
var o, err5 = bbs.CompleteMultipartUpload(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_CompleteMultipartUploadResponse(*o)
if s.Expiration != nil {
ho.Add("x-amz-expiration", string(*s.Expiration))}
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_CopyObject(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.CopyObjectInput{}
if len(hi.Values("x-amz-source-expected-bucket-owner")) != 0 {
i.ExpectedSourceBucketOwner = thing_pointer(hi.Get("x-amz-source-expected-bucket-owner"))}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-object-lock-legal-hold")) != 0 {
var s = hi.Get("x-amz-object-lock-legal-hold")
var x, err2 = intern_ObjectLockLegalHoldStatus(s)
if err2 != nil {input_errors["x-amz-object-lock-legal-hold"] = &Bb_input_error{"x-amz-object-lock-legal-hold", err2}} else {i.ObjectLockLegalHoldStatus = x}}
if len(hi.Values("x-amz-object-lock-retain-until-date")) != 0 {
var s = hi.Get("x-amz-object-lock-retain-until-date")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-object-lock-retain-until-date"] = &Bb_input_error{"x-amz-object-lock-retain-until-date", err2}} else {i.ObjectLockRetainUntilDate = &x}}
if len(hi.Values("x-amz-object-lock-mode")) != 0 {
var s = hi.Get("x-amz-object-lock-mode")
var x, err2 = intern_ObjectLockMode(s)
if err2 != nil {input_errors["x-amz-object-lock-mode"] = &Bb_input_error{"x-amz-object-lock-mode", err2}} else {i.ObjectLockMode = x}}
if len(hi.Values("x-amz-tagging")) != 0 {
i.Tagging = thing_pointer(hi.Get("x-amz-tagging"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-copy-source-server-side-encryption-customer-key-MD5")) != 0 {
i.CopySourceSSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-copy-source-server-side-encryption-customer-key")) != 0 {
i.CopySourceSSECustomerKey = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-copy-source-server-side-encryption-customer-algorithm")) != 0 {
i.CopySourceSSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-algorithm"))}
if len(hi.Values("x-amz-server-side-encryption-bucket-key-enabled")) != 0 {
var s = hi.Get("x-amz-server-side-encryption-bucket-key-enabled")
var x, err2 = strconv.ParseBool(s)
if err2 != nil {input_errors["x-amz-server-side-encryption-bucket-key-enabled"] = &Bb_input_error{"x-amz-server-side-encryption-bucket-key-enabled", err2}} else {i.BucketKeyEnabled = &x}}
if len(hi.Values("x-amz-server-side-encryption-context")) != 0 {
i.SSEKMSEncryptionContext = thing_pointer(hi.Get("x-amz-server-side-encryption-context"))}
if len(hi.Values("x-amz-server-side-encryption-aws-kms-key-id")) != 0 {
i.SSEKMSKeyId = thing_pointer(hi.Get("x-amz-server-side-encryption-aws-kms-key-id"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if len(hi.Values("x-amz-website-redirect-location")) != 0 {
i.WebsiteRedirectLocation = thing_pointer(hi.Get("x-amz-website-redirect-location"))}
if len(hi.Values("x-amz-storage-class")) != 0 {
var s = hi.Get("x-amz-storage-class")
var x, err2 = intern_StorageClass(s)
if err2 != nil {input_errors["x-amz-storage-class"] = &Bb_input_error{"x-amz-storage-class", err2}} else {i.StorageClass = x}}
if len(hi.Values("x-amz-server-side-encryption")) != 0 {
var s = hi.Get("x-amz-server-side-encryption")
var x, err2 = intern_ServerSideEncryption(s)
if err2 != nil {input_errors["x-amz-server-side-encryption"] = &Bb_input_error{"x-amz-server-side-encryption", err2}} else {i.ServerSideEncryption = x}}
if len(hi.Values("x-amz-tagging-directive")) != 0 {
var s = hi.Get("x-amz-tagging-directive")
var x, err2 = intern_TaggingDirective(s)
if err2 != nil {input_errors["x-amz-tagging-directive"] = &Bb_input_error{"x-amz-tagging-directive", err2}} else {i.TaggingDirective = x}}
if len(hi.Values("x-amz-metadata-directive")) != 0 {
var s = hi.Get("x-amz-metadata-directive")
var x, err2 = intern_MetadataDirective(s)
if err2 != nil {input_errors["x-amz-metadata-directive"] = &Bb_input_error{"x-amz-metadata-directive", err2}} else {i.MetadataDirective = x}}
if len(hi.Values("x-amz-meta-")) != 0 {
var prefix = http.CanonicalHeaderKey("x-amz-meta-")
var bin map[string]string
for k, v := range hi {
if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}
i.Metadata = bin}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
if len(hi.Values("x-amz-grant-write-acp")) != 0 {
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))}
if len(hi.Values("x-amz-grant-read-acp")) != 0 {
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))}
if len(hi.Values("x-amz-grant-read")) != 0 {
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))}
if len(hi.Values("x-amz-grant-full-control")) != 0 {
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))}
if len(hi.Values("Expires")) != 0 {
var s = hi.Get("Expires")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["Expires"] = &Bb_input_error{"Expires", err2}} else {i.Expires = &x}}
if len(hi.Values("x-amz-copy-source-if-unmodified-since")) != 0 {
var s = hi.Get("x-amz-copy-source-if-unmodified-since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-copy-source-if-unmodified-since"] = &Bb_input_error{"x-amz-copy-source-if-unmodified-since", err2}} else {i.CopySourceIfUnmodifiedSince = &x}}
if len(hi.Values("x-amz-copy-source-if-none-match")) != 0 {
i.CopySourceIfNoneMatch = thing_pointer(hi.Get("x-amz-copy-source-if-none-match"))}
if len(hi.Values("x-amz-copy-source-if-modified-since")) != 0 {
var s = hi.Get("x-amz-copy-source-if-modified-since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-copy-source-if-modified-since"] = &Bb_input_error{"x-amz-copy-source-if-modified-since", err2}} else {i.CopySourceIfModifiedSince = &x}}
if len(hi.Values("x-amz-copy-source-if-match")) != 0 {
i.CopySourceIfMatch = thing_pointer(hi.Get("x-amz-copy-source-if-match"))}
if len(hi.Values("x-amz-copy-source")) != 0 {
i.CopySource = thing_pointer(hi.Get("x-amz-copy-source"))}
if len(hi.Values("Content-Type")) != 0 {
i.ContentType = thing_pointer(hi.Get("Content-Type"))}
if len(hi.Values("Content-Language")) != 0 {
i.ContentLanguage = thing_pointer(hi.Get("Content-Language"))}
if len(hi.Values("Content-Encoding")) != 0 {
i.ContentEncoding = thing_pointer(hi.Get("Content-Encoding"))}
if len(hi.Values("Content-Disposition")) != 0 {
i.ContentDisposition = thing_pointer(hi.Get("Content-Disposition"))}
if len(hi.Values("x-amz-checksum-algorithm")) != 0 {
var s = hi.Get("x-amz-checksum-algorithm")
var x, err2 = intern_ChecksumAlgorithm(s)
if err2 != nil {input_errors["x-amz-checksum-algorithm"] = &Bb_input_error{"x-amz-checksum-algorithm", err2}} else {i.ChecksumAlgorithm = x}}
if len(hi.Values("Cache-Control")) != 0 {
i.CacheControl = thing_pointer(hi.Get("Cache-Control"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
if len(hi.Values("x-amz-acl")) != 0 {
var s = hi.Get("x-amz-acl")
var x, err2 = intern_ObjectCannedACL(s)
if err2 != nil {input_errors["x-amz-acl"] = &Bb_input_error{"x-amz-acl", err2}} else {i.ACL = x}}
var o, err5 = bbs.CopyObject(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_CopyObjectResponse(*o)
if s.Expiration != nil {
ho.Add("x-amz-expiration", string(*s.Expiration))}
if s.CopySourceVersionId != nil {
ho.Add("x-amz-copy-source-version-id", string(*s.CopySourceVersionId))}
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
if s.SSECustomerAlgorithm != nil {
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))}
if s.SSECustomerKeyMD5 != nil {
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.SSEKMSEncryptionContext != nil {
ho.Add("x-amz-server-side-encryption-context", string(*s.SSEKMSEncryptionContext))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_CreateBucket(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.CreateBucketInput{}
if len(hi.Values("x-amz-object-ownership")) != 0 {
var s = hi.Get("x-amz-object-ownership")
var x, err2 = intern_ObjectOwnership(s)
if err2 != nil {input_errors["x-amz-object-ownership"] = &Bb_input_error{"x-amz-object-ownership", err2}} else {i.ObjectOwnership = x}}
if len(hi.Values("x-amz-bucket-object-lock-enabled")) != 0 {
var s = hi.Get("x-amz-bucket-object-lock-enabled")
var x, err2 = strconv.ParseBool(s)
if err2 != nil {input_errors["x-amz-bucket-object-lock-enabled"] = &Bb_input_error{"x-amz-bucket-object-lock-enabled", err2}} else {i.ObjectLockEnabledForBucket = &x}}
if len(hi.Values("x-amz-grant-write-acp")) != 0 {
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))}
if len(hi.Values("x-amz-grant-write")) != 0 {
i.GrantWrite = thing_pointer(hi.Get("x-amz-grant-write"))}
if len(hi.Values("x-amz-grant-read-acp")) != 0 {
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))}
if len(hi.Values("x-amz-grant-read")) != 0 {
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))}
if len(hi.Values("x-amz-grant-full-control")) != 0 {
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
if len(hi.Values("x-amz-acl")) != 0 {
var s = hi.Get("x-amz-acl")
var x, err2 = intern_BucketCannedACL(s)
if err2 != nil {input_errors["x-amz-acl"] = &Bb_input_error{"x-amz-acl", err2}} else {i.ACL = x}}
{var x types.CreateBucketConfiguration
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {panic(fmt.Errorf("No http body for types.CreateBucketConfiguration: %w", err1))}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {panic(fmt.Errorf("Invalid http body for types.CreateBucketConfiguration: %w", err2))}
i.CreateBucketConfiguration = &x}
var o, err5 = bbs.CreateBucket(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_CreateBucketResponse(*o)
if s.Location != nil {
ho.Add("Location", string(*s.Location))}
if s.BucketArn != nil {
ho.Add("x-amz-bucket-arn", string(*s.BucketArn))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_CreateMultipartUpload(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.CreateMultipartUploadInput{}
if len(hi.Values("x-amz-checksum-type")) != 0 {
var s = hi.Get("x-amz-checksum-type")
var x, err2 = intern_ChecksumType(s)
if err2 != nil {input_errors["x-amz-checksum-type"] = &Bb_input_error{"x-amz-checksum-type", err2}} else {i.ChecksumType = x}}
if len(hi.Values("x-amz-checksum-algorithm")) != 0 {
var s = hi.Get("x-amz-checksum-algorithm")
var x, err2 = intern_ChecksumAlgorithm(s)
if err2 != nil {input_errors["x-amz-checksum-algorithm"] = &Bb_input_error{"x-amz-checksum-algorithm", err2}} else {i.ChecksumAlgorithm = x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-object-lock-legal-hold")) != 0 {
var s = hi.Get("x-amz-object-lock-legal-hold")
var x, err2 = intern_ObjectLockLegalHoldStatus(s)
if err2 != nil {input_errors["x-amz-object-lock-legal-hold"] = &Bb_input_error{"x-amz-object-lock-legal-hold", err2}} else {i.ObjectLockLegalHoldStatus = x}}
if len(hi.Values("x-amz-object-lock-retain-until-date")) != 0 {
var s = hi.Get("x-amz-object-lock-retain-until-date")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-object-lock-retain-until-date"] = &Bb_input_error{"x-amz-object-lock-retain-until-date", err2}} else {i.ObjectLockRetainUntilDate = &x}}
if len(hi.Values("x-amz-object-lock-mode")) != 0 {
var s = hi.Get("x-amz-object-lock-mode")
var x, err2 = intern_ObjectLockMode(s)
if err2 != nil {input_errors["x-amz-object-lock-mode"] = &Bb_input_error{"x-amz-object-lock-mode", err2}} else {i.ObjectLockMode = x}}
if len(hi.Values("x-amz-tagging")) != 0 {
i.Tagging = thing_pointer(hi.Get("x-amz-tagging"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-server-side-encryption-bucket-key-enabled")) != 0 {
var s = hi.Get("x-amz-server-side-encryption-bucket-key-enabled")
var x, err2 = strconv.ParseBool(s)
if err2 != nil {input_errors["x-amz-server-side-encryption-bucket-key-enabled"] = &Bb_input_error{"x-amz-server-side-encryption-bucket-key-enabled", err2}} else {i.BucketKeyEnabled = &x}}
if len(hi.Values("x-amz-server-side-encryption-context")) != 0 {
i.SSEKMSEncryptionContext = thing_pointer(hi.Get("x-amz-server-side-encryption-context"))}
if len(hi.Values("x-amz-server-side-encryption-aws-kms-key-id")) != 0 {
i.SSEKMSKeyId = thing_pointer(hi.Get("x-amz-server-side-encryption-aws-kms-key-id"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if len(hi.Values("x-amz-website-redirect-location")) != 0 {
i.WebsiteRedirectLocation = thing_pointer(hi.Get("x-amz-website-redirect-location"))}
if len(hi.Values("x-amz-storage-class")) != 0 {
var s = hi.Get("x-amz-storage-class")
var x, err2 = intern_StorageClass(s)
if err2 != nil {input_errors["x-amz-storage-class"] = &Bb_input_error{"x-amz-storage-class", err2}} else {i.StorageClass = x}}
if len(hi.Values("x-amz-server-side-encryption")) != 0 {
var s = hi.Get("x-amz-server-side-encryption")
var x, err2 = intern_ServerSideEncryption(s)
if err2 != nil {input_errors["x-amz-server-side-encryption"] = &Bb_input_error{"x-amz-server-side-encryption", err2}} else {i.ServerSideEncryption = x}}
if len(hi.Values("x-amz-meta-")) != 0 {
var prefix = http.CanonicalHeaderKey("x-amz-meta-")
var bin map[string]string
for k, v := range hi {
if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}
i.Metadata = bin}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
if len(hi.Values("x-amz-grant-write-acp")) != 0 {
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))}
if len(hi.Values("x-amz-grant-read-acp")) != 0 {
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))}
if len(hi.Values("x-amz-grant-read")) != 0 {
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))}
if len(hi.Values("x-amz-grant-full-control")) != 0 {
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))}
if len(hi.Values("Expires")) != 0 {
var s = hi.Get("Expires")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["Expires"] = &Bb_input_error{"Expires", err2}} else {i.Expires = &x}}
if len(hi.Values("Content-Type")) != 0 {
i.ContentType = thing_pointer(hi.Get("Content-Type"))}
if len(hi.Values("Content-Language")) != 0 {
i.ContentLanguage = thing_pointer(hi.Get("Content-Language"))}
if len(hi.Values("Content-Encoding")) != 0 {
i.ContentEncoding = thing_pointer(hi.Get("Content-Encoding"))}
if len(hi.Values("Content-Disposition")) != 0 {
i.ContentDisposition = thing_pointer(hi.Get("Content-Disposition"))}
if len(hi.Values("Cache-Control")) != 0 {
i.CacheControl = thing_pointer(hi.Get("Cache-Control"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
if len(hi.Values("x-amz-acl")) != 0 {
var s = hi.Get("x-amz-acl")
var x, err2 = intern_ObjectCannedACL(s)
if err2 != nil {input_errors["x-amz-acl"] = &Bb_input_error{"x-amz-acl", err2}} else {i.ACL = x}}
var o, err5 = bbs.CreateMultipartUpload(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_CreateMultipartUploadResponse(*o)
if s.AbortDate != nil {
ho.Add("x-amz-abort-date", s.AbortDate.String())}
if s.AbortRuleId != nil {
ho.Add("x-amz-abort-rule-id", string(*s.AbortRuleId))}
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
if s.SSECustomerAlgorithm != nil {
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))}
if s.SSECustomerKeyMD5 != nil {
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.SSEKMSEncryptionContext != nil {
ho.Add("x-amz-server-side-encryption-context", string(*s.SSEKMSEncryptionContext))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
if s.ChecksumAlgorithm != "" {
ho.Add("x-amz-checksum-algorithm", string(s.ChecksumAlgorithm))}
if s.ChecksumType != "" {
ho.Add("x-amz-checksum-type", string(s.ChecksumType))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_DeleteBucket(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.DeleteBucketInput{}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var _, err5 = bbs.DeleteBucket(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
ho.Set("Content-Type", "application/xml")
var status int = 204
w.WriteHeader(status)
}
func h_DeleteObject(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.DeleteObjectInput{}
if len(hi.Values("x-amz-if-match-size")) != 0 {
var s = hi.Get("x-amz-if-match-size")
var x, err2 = strconv.ParseInt(s, 10, 64)
if err2 != nil {input_errors["x-amz-if-match-size"] = &Bb_input_error{"x-amz-if-match-size", err2}} else {i.IfMatchSize = &x}}
if len(hi.Values("x-amz-if-match-last-modified-time")) != 0 {
var s = hi.Get("x-amz-if-match-last-modified-time")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-if-match-last-modified-time"] = &Bb_input_error{"x-amz-if-match-last-modified-time", err2}} else {i.IfMatchLastModifiedTime = &x}}
if len(hi.Values("If-Match")) != 0 {
i.IfMatch = thing_pointer(hi.Get("If-Match"))}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-bypass-governance-retention")) != 0 {
var s = hi.Get("x-amz-bypass-governance-retention")
var x, err2 = strconv.ParseBool(s)
if err2 != nil {input_errors["x-amz-bypass-governance-retention"] = &Bb_input_error{"x-amz-bypass-governance-retention", err2}} else {i.BypassGovernanceRetention = &x}}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if qi.Has("versionId") {
i.VersionId = thing_pointer(qi.Get("versionId"))}
if len(hi.Values("x-amz-mfa")) != 0 {
i.MFA = thing_pointer(hi.Get("x-amz-mfa"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.DeleteObject(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_DeleteObjectResponse(*o)
if s.DeleteMarker != nil {
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))}
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 204
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_DeleteObjects(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.DeleteObjectsInput{}
if len(hi.Values("x-amz-sdk-checksum-algorithm")) != 0 {
var s = hi.Get("x-amz-sdk-checksum-algorithm")
var x, err2 = intern_ChecksumAlgorithm(s)
if err2 != nil {input_errors["x-amz-sdk-checksum-algorithm"] = &Bb_input_error{"x-amz-sdk-checksum-algorithm", err2}} else {i.ChecksumAlgorithm = x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-bypass-governance-retention")) != 0 {
var s = hi.Get("x-amz-bypass-governance-retention")
var x, err2 = strconv.ParseBool(s)
if err2 != nil {input_errors["x-amz-bypass-governance-retention"] = &Bb_input_error{"x-amz-bypass-governance-retention", err2}} else {i.BypassGovernanceRetention = &x}}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-mfa")) != 0 {
i.MFA = thing_pointer(hi.Get("x-amz-mfa"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
{var x types.Delete
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {panic(fmt.Errorf("No http body for types.Delete: %w", err1))}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {panic(fmt.Errorf("Invalid http body for types.Delete: %w", err2))}
i.Delete = &x}
var o, err5 = bbs.DeleteObjects(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_DeleteObjectsResponse(*o)
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_DeleteObjectTagging(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.DeleteObjectTaggingInput{}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if qi.Has("versionId") {
i.VersionId = thing_pointer(qi.Get("versionId"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.DeleteObjectTagging(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_DeleteObjectTaggingResponse(*o)
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 204
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_GetObject(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.GetObjectInput{}
if len(hi.Values("x-amz-checksum-mode")) != 0 {
var s = hi.Get("x-amz-checksum-mode")
var x, err2 = intern_ChecksumMode(s)
if err2 != nil {input_errors["x-amz-checksum-mode"] = &Bb_input_error{"x-amz-checksum-mode", err2}} else {i.ChecksumMode = x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if qi.Has("partNumber") {
var s = qi.Get("partNumber")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["partNumber"] = &Bb_input_error{"partNumber", err2}} else {i.PartNumber = &x2}}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if qi.Has("versionId") {
i.VersionId = thing_pointer(qi.Get("versionId"))}
if qi.Has("response-expires") {
var s = qi.Get("response-expires")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["response-expires"] = &Bb_input_error{"response-expires", err2}} else {i.ResponseExpires = &x}}
if qi.Has("response-content-type") {
i.ResponseContentType = thing_pointer(qi.Get("response-content-type"))}
if qi.Has("response-content-language") {
i.ResponseContentLanguage = thing_pointer(qi.Get("response-content-language"))}
if qi.Has("response-content-encoding") {
i.ResponseContentEncoding = thing_pointer(qi.Get("response-content-encoding"))}
if qi.Has("response-content-disposition") {
i.ResponseContentDisposition = thing_pointer(qi.Get("response-content-disposition"))}
if qi.Has("response-cache-control") {
i.ResponseCacheControl = thing_pointer(qi.Get("response-cache-control"))}
if len(hi.Values("Range")) != 0 {
i.Range = thing_pointer(hi.Get("Range"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
if len(hi.Values("If-Unmodified-Since")) != 0 {
var s = hi.Get("If-Unmodified-Since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["If-Unmodified-Since"] = &Bb_input_error{"If-Unmodified-Since", err2}} else {i.IfUnmodifiedSince = &x}}
if len(hi.Values("If-None-Match")) != 0 {
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))}
if len(hi.Values("If-Modified-Since")) != 0 {
var s = hi.Get("If-Modified-Since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["If-Modified-Since"] = &Bb_input_error{"If-Modified-Since", err2}} else {i.IfModifiedSince = &x}}
if len(hi.Values("If-Match")) != 0 {
i.IfMatch = thing_pointer(hi.Get("If-Match"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.GetObject(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_GetObjectResponse(*o)
if s.DeleteMarker != nil {
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))}
if s.AcceptRanges != nil {
ho.Add("accept-ranges", string(*s.AcceptRanges))}
if s.Expiration != nil {
ho.Add("x-amz-expiration", string(*s.Expiration))}
if s.Restore != nil {
ho.Add("x-amz-restore", string(*s.Restore))}
if s.LastModified != nil {
ho.Add("Last-Modified", s.LastModified.String())}
if s.ContentLength != nil {
ho.Add("Content-Length", strconv.FormatInt(*s.ContentLength, 10))}
if s.ETag != nil {
ho.Add("ETag", string(*s.ETag))}
if s.ChecksumCRC32 != nil {
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))}
if s.ChecksumCRC32C != nil {
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))}
if s.ChecksumCRC64NVME != nil {
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))}
if s.ChecksumSHA1 != nil {
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))}
if s.ChecksumSHA256 != nil {
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))}
if s.ChecksumType != "" {
ho.Add("x-amz-checksum-type", string(s.ChecksumType))}
if s.MissingMeta != nil {
ho.Add("x-amz-missing-meta", strconv.FormatInt(int64(*s.MissingMeta), 10))}
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
if s.CacheControl != nil {
ho.Add("Cache-Control", string(*s.CacheControl))}
if s.ContentDisposition != nil {
ho.Add("Content-Disposition", string(*s.ContentDisposition))}
if s.ContentEncoding != nil {
ho.Add("Content-Encoding", string(*s.ContentEncoding))}
if s.ContentLanguage != nil {
ho.Add("Content-Language", string(*s.ContentLanguage))}
if s.ContentRange != nil {
ho.Add("Content-Range", string(*s.ContentRange))}
if s.ContentType != nil {
ho.Add("Content-Type", string(*s.ContentType))}
if s.Expires != nil {
ho.Add("Expires", s.Expires.String())}
if s.WebsiteRedirectLocation != nil {
ho.Add("x-amz-website-redirect-location", string(*s.WebsiteRedirectLocation))}
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
for k, v := range s.Metadata {
ho.Add(k, v)}
if s.SSECustomerAlgorithm != nil {
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))}
if s.SSECustomerKeyMD5 != nil {
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.StorageClass != "" {
ho.Add("x-amz-storage-class", string(s.StorageClass))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
if s.ReplicationStatus != "" {
ho.Add("x-amz-replication-status", string(s.ReplicationStatus))}
if s.PartsCount != nil {
ho.Add("x-amz-mp-parts-count", strconv.FormatInt(int64(*s.PartsCount), 10))}
if s.TagCount != nil {
ho.Add("x-amz-tagging-count", strconv.FormatInt(int64(*s.TagCount), 10))}
if s.ObjectLockMode != "" {
ho.Add("x-amz-object-lock-mode", string(s.ObjectLockMode))}
if s.ObjectLockRetainUntilDate != nil {
ho.Add("x-amz-object-lock-retain-until-date", s.ObjectLockRetainUntilDate.String())}
if s.ObjectLockLegalHoldStatus != "" {
ho.Add("x-amz-object-lock-legal-hold", string(s.ObjectLockLegalHoldStatus))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_GetObjectAttributes(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.GetObjectAttributesInput{}
if len(hi.Values("x-amz-object-attributes")) != 0 {
var rhs = hi.Values("x-amz-object-attributes")
var bin []types.ObjectAttributes
for _, v := range slices.All(rhs) {
var s = v
var x, err2 = intern_ObjectAttributes(s)
if err2 != nil {input_errors["x-amz-object-attributes"] = &Bb_input_error{"x-amz-object-attributes", err2}} else {bin = append(bin, x)}}
i.ObjectAttributes = bin}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if len(hi.Values("x-amz-part-number-marker")) != 0 {
i.PartNumberMarker = thing_pointer(hi.Get("x-amz-part-number-marker"))}
if len(hi.Values("x-amz-max-parts")) != 0 {
var s = hi.Get("x-amz-max-parts")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["x-amz-max-parts"] = &Bb_input_error{"x-amz-max-parts", err2}} else {i.MaxParts = &x2}}
if qi.Has("versionId") {
i.VersionId = thing_pointer(qi.Get("versionId"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.GetObjectAttributes(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_GetObjectAttributesResponse(*o)
if s.DeleteMarker != nil {
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))}
if s.LastModified != nil {
ho.Add("Last-Modified", s.LastModified.String())}
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_GetObjectTagging(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.GetObjectTaggingInput{}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if qi.Has("versionId") {
i.VersionId = thing_pointer(qi.Get("versionId"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.GetObjectTagging(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_GetObjectTaggingResponse(*o)
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_HeadBucket(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.HeadBucketInput{}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.HeadBucket(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_HeadBucketResponse(*o)
if s.BucketArn != nil {
ho.Add("x-amz-bucket-arn", string(*s.BucketArn))}
if s.BucketLocationType != "" {
ho.Add("x-amz-bucket-location-type", string(s.BucketLocationType))}
if s.BucketLocationName != nil {
ho.Add("x-amz-bucket-location-name", string(*s.BucketLocationName))}
if s.BucketRegion != nil {
ho.Add("x-amz-bucket-region", string(*s.BucketRegion))}
if s.AccessPointAlias != nil {
ho.Add("x-amz-access-point-alias", strconv.FormatBool(*s.AccessPointAlias))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_HeadObject(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.HeadObjectInput{}
if len(hi.Values("x-amz-checksum-mode")) != 0 {
var s = hi.Get("x-amz-checksum-mode")
var x, err2 = intern_ChecksumMode(s)
if err2 != nil {input_errors["x-amz-checksum-mode"] = &Bb_input_error{"x-amz-checksum-mode", err2}} else {i.ChecksumMode = x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if qi.Has("partNumber") {
var s = qi.Get("partNumber")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["partNumber"] = &Bb_input_error{"partNumber", err2}} else {i.PartNumber = &x2}}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if qi.Has("versionId") {
i.VersionId = thing_pointer(qi.Get("versionId"))}
if qi.Has("response-expires") {
var s = qi.Get("response-expires")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["response-expires"] = &Bb_input_error{"response-expires", err2}} else {i.ResponseExpires = &x}}
if qi.Has("response-content-type") {
i.ResponseContentType = thing_pointer(qi.Get("response-content-type"))}
if qi.Has("response-content-language") {
i.ResponseContentLanguage = thing_pointer(qi.Get("response-content-language"))}
if qi.Has("response-content-encoding") {
i.ResponseContentEncoding = thing_pointer(qi.Get("response-content-encoding"))}
if qi.Has("response-content-disposition") {
i.ResponseContentDisposition = thing_pointer(qi.Get("response-content-disposition"))}
if qi.Has("response-cache-control") {
i.ResponseCacheControl = thing_pointer(qi.Get("response-cache-control"))}
if len(hi.Values("Range")) != 0 {
i.Range = thing_pointer(hi.Get("Range"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
if len(hi.Values("If-Unmodified-Since")) != 0 {
var s = hi.Get("If-Unmodified-Since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["If-Unmodified-Since"] = &Bb_input_error{"If-Unmodified-Since", err2}} else {i.IfUnmodifiedSince = &x}}
if len(hi.Values("If-None-Match")) != 0 {
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))}
if len(hi.Values("If-Modified-Since")) != 0 {
var s = hi.Get("If-Modified-Since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["If-Modified-Since"] = &Bb_input_error{"If-Modified-Since", err2}} else {i.IfModifiedSince = &x}}
if len(hi.Values("If-Match")) != 0 {
i.IfMatch = thing_pointer(hi.Get("If-Match"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.HeadObject(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_HeadObjectResponse(*o)
if s.DeleteMarker != nil {
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))}
if s.AcceptRanges != nil {
ho.Add("accept-ranges", string(*s.AcceptRanges))}
if s.Expiration != nil {
ho.Add("x-amz-expiration", string(*s.Expiration))}
if s.Restore != nil {
ho.Add("x-amz-restore", string(*s.Restore))}
if s.ArchiveStatus != "" {
ho.Add("x-amz-archive-status", string(s.ArchiveStatus))}
if s.LastModified != nil {
ho.Add("Last-Modified", s.LastModified.String())}
if s.ContentLength != nil {
ho.Add("Content-Length", strconv.FormatInt(*s.ContentLength, 10))}
if s.ChecksumCRC32 != nil {
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))}
if s.ChecksumCRC32C != nil {
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))}
if s.ChecksumCRC64NVME != nil {
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))}
if s.ChecksumSHA1 != nil {
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))}
if s.ChecksumSHA256 != nil {
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))}
if s.ChecksumType != "" {
ho.Add("x-amz-checksum-type", string(s.ChecksumType))}
if s.ETag != nil {
ho.Add("ETag", string(*s.ETag))}
if s.MissingMeta != nil {
ho.Add("x-amz-missing-meta", strconv.FormatInt(int64(*s.MissingMeta), 10))}
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
if s.CacheControl != nil {
ho.Add("Cache-Control", string(*s.CacheControl))}
if s.ContentDisposition != nil {
ho.Add("Content-Disposition", string(*s.ContentDisposition))}
if s.ContentEncoding != nil {
ho.Add("Content-Encoding", string(*s.ContentEncoding))}
if s.ContentLanguage != nil {
ho.Add("Content-Language", string(*s.ContentLanguage))}
if s.ContentType != nil {
ho.Add("Content-Type", string(*s.ContentType))}
if s.ContentRange != nil {
ho.Add("Content-Range", string(*s.ContentRange))}
if s.Expires != nil {
ho.Add("Expires", s.Expires.String())}
if s.WebsiteRedirectLocation != nil {
ho.Add("x-amz-website-redirect-location", string(*s.WebsiteRedirectLocation))}
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
for k, v := range s.Metadata {
ho.Add(k, v)}
if s.SSECustomerAlgorithm != nil {
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))}
if s.SSECustomerKeyMD5 != nil {
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.StorageClass != "" {
ho.Add("x-amz-storage-class", string(s.StorageClass))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
if s.ReplicationStatus != "" {
ho.Add("x-amz-replication-status", string(s.ReplicationStatus))}
if s.PartsCount != nil {
ho.Add("x-amz-mp-parts-count", strconv.FormatInt(int64(*s.PartsCount), 10))}
if s.TagCount != nil {
ho.Add("x-amz-tagging-count", strconv.FormatInt(int64(*s.TagCount), 10))}
if s.ObjectLockMode != "" {
ho.Add("x-amz-object-lock-mode", string(s.ObjectLockMode))}
if s.ObjectLockRetainUntilDate != nil {
ho.Add("x-amz-object-lock-retain-until-date", s.ObjectLockRetainUntilDate.String())}
if s.ObjectLockLegalHoldStatus != "" {
ho.Add("x-amz-object-lock-legal-hold", string(s.ObjectLockLegalHoldStatus))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_ListBuckets(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.ListBucketsInput{}
if qi.Has("bucket-region") {
i.BucketRegion = thing_pointer(qi.Get("bucket-region"))}
if qi.Has("prefix") {
i.Prefix = thing_pointer(qi.Get("prefix"))}
if qi.Has("continuation-token") {
i.ContinuationToken = thing_pointer(qi.Get("continuation-token"))}
if qi.Has("max-buckets") {
var s = qi.Get("max-buckets")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["max-buckets"] = &Bb_input_error{"max-buckets", err2}} else {i.MaxBuckets = &x2}}
var o, err5 = bbs.ListBuckets(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_ListBucketsResponse(*o)
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_ListMultipartUploads(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.ListMultipartUploadsInput{}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if qi.Has("upload-id-marker") {
i.UploadIdMarker = thing_pointer(qi.Get("upload-id-marker"))}
if qi.Has("prefix") {
i.Prefix = thing_pointer(qi.Get("prefix"))}
if qi.Has("max-uploads") {
var s = qi.Get("max-uploads")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["max-uploads"] = &Bb_input_error{"max-uploads", err2}} else {i.MaxUploads = &x2}}
if qi.Has("key-marker") {
i.KeyMarker = thing_pointer(qi.Get("key-marker"))}
if qi.Has("encoding-type") {
var s = qi.Get("encoding-type")
var x, err2 = intern_EncodingType(s)
if err2 != nil {input_errors["encoding-type"] = &Bb_input_error{"encoding-type", err2}} else {i.EncodingType = x}}
if qi.Has("delimiter") {
i.Delimiter = thing_pointer(qi.Get("delimiter"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.ListMultipartUploads(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_ListMultipartUploadsResponse(*o)
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_ListObjects(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.ListObjectsInput{}
if len(hi.Values("x-amz-optional-object-attributes")) != 0 {
var rhs = hi.Values("x-amz-optional-object-attributes")
var bin []types.OptionalObjectAttributes
for _, v := range slices.All(rhs) {
var s = v
var x, err2 = intern_OptionalObjectAttributes(s)
if err2 != nil {input_errors["x-amz-optional-object-attributes"] = &Bb_input_error{"x-amz-optional-object-attributes", err2}} else {bin = append(bin, x)}}
i.OptionalObjectAttributes = bin}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if qi.Has("prefix") {
i.Prefix = thing_pointer(qi.Get("prefix"))}
if qi.Has("max-keys") {
var s = qi.Get("max-keys")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["max-keys"] = &Bb_input_error{"max-keys", err2}} else {i.MaxKeys = &x2}}
if qi.Has("marker") {
i.Marker = thing_pointer(qi.Get("marker"))}
if qi.Has("encoding-type") {
var s = qi.Get("encoding-type")
var x, err2 = intern_EncodingType(s)
if err2 != nil {input_errors["encoding-type"] = &Bb_input_error{"encoding-type", err2}} else {i.EncodingType = x}}
if qi.Has("delimiter") {
i.Delimiter = thing_pointer(qi.Get("delimiter"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.ListObjects(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_ListObjectsResponse(*o)
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_ListObjectsV2(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.ListObjectsV2Input{}
if len(hi.Values("x-amz-optional-object-attributes")) != 0 {
var rhs = hi.Values("x-amz-optional-object-attributes")
var bin []types.OptionalObjectAttributes
for _, v := range slices.All(rhs) {
var s = v
var x, err2 = intern_OptionalObjectAttributes(s)
if err2 != nil {input_errors["x-amz-optional-object-attributes"] = &Bb_input_error{"x-amz-optional-object-attributes", err2}} else {bin = append(bin, x)}}
i.OptionalObjectAttributes = bin}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if qi.Has("start-after") {
i.StartAfter = thing_pointer(qi.Get("start-after"))}
if qi.Has("fetch-owner") {
var s = qi.Get("fetch-owner")
var x, err2 = strconv.ParseBool(s)
if err2 != nil {input_errors["fetch-owner"] = &Bb_input_error{"fetch-owner", err2}} else {i.FetchOwner = &x}}
if qi.Has("continuation-token") {
i.ContinuationToken = thing_pointer(qi.Get("continuation-token"))}
if qi.Has("prefix") {
i.Prefix = thing_pointer(qi.Get("prefix"))}
if qi.Has("max-keys") {
var s = qi.Get("max-keys")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["max-keys"] = &Bb_input_error{"max-keys", err2}} else {i.MaxKeys = &x2}}
if qi.Has("encoding-type") {
var s = qi.Get("encoding-type")
var x, err2 = intern_EncodingType(s)
if err2 != nil {input_errors["encoding-type"] = &Bb_input_error{"encoding-type", err2}} else {i.EncodingType = x}}
if qi.Has("delimiter") {
i.Delimiter = thing_pointer(qi.Get("delimiter"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.ListObjectsV2(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_ListObjectsV2Response(*o)
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_ListParts(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.ListPartsInput{}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if qi.Has("uploadId") {
i.UploadId = thing_pointer(qi.Get("uploadId"))}
if qi.Has("part-number-marker") {
i.PartNumberMarker = thing_pointer(qi.Get("part-number-marker"))}
if qi.Has("max-parts") {
var s = qi.Get("max-parts")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["max-parts"] = &Bb_input_error{"max-parts", err2}} else {i.MaxParts = &x2}}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.ListParts(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_ListPartsResponse(*o)
if s.AbortDate != nil {
ho.Add("x-amz-abort-date", s.AbortDate.String())}
if s.AbortRuleId != nil {
ho.Add("x-amz-abort-rule-id", string(*s.AbortRuleId))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_PutObject(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.PutObjectInput{}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-object-lock-legal-hold")) != 0 {
var s = hi.Get("x-amz-object-lock-legal-hold")
var x, err2 = intern_ObjectLockLegalHoldStatus(s)
if err2 != nil {input_errors["x-amz-object-lock-legal-hold"] = &Bb_input_error{"x-amz-object-lock-legal-hold", err2}} else {i.ObjectLockLegalHoldStatus = x}}
if len(hi.Values("x-amz-object-lock-retain-until-date")) != 0 {
var s = hi.Get("x-amz-object-lock-retain-until-date")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-object-lock-retain-until-date"] = &Bb_input_error{"x-amz-object-lock-retain-until-date", err2}} else {i.ObjectLockRetainUntilDate = &x}}
if len(hi.Values("x-amz-object-lock-mode")) != 0 {
var s = hi.Get("x-amz-object-lock-mode")
var x, err2 = intern_ObjectLockMode(s)
if err2 != nil {input_errors["x-amz-object-lock-mode"] = &Bb_input_error{"x-amz-object-lock-mode", err2}} else {i.ObjectLockMode = x}}
if len(hi.Values("x-amz-tagging")) != 0 {
i.Tagging = thing_pointer(hi.Get("x-amz-tagging"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-server-side-encryption-bucket-key-enabled")) != 0 {
var s = hi.Get("x-amz-server-side-encryption-bucket-key-enabled")
var x, err2 = strconv.ParseBool(s)
if err2 != nil {input_errors["x-amz-server-side-encryption-bucket-key-enabled"] = &Bb_input_error{"x-amz-server-side-encryption-bucket-key-enabled", err2}} else {i.BucketKeyEnabled = &x}}
if len(hi.Values("x-amz-server-side-encryption-context")) != 0 {
i.SSEKMSEncryptionContext = thing_pointer(hi.Get("x-amz-server-side-encryption-context"))}
if len(hi.Values("x-amz-server-side-encryption-aws-kms-key-id")) != 0 {
i.SSEKMSKeyId = thing_pointer(hi.Get("x-amz-server-side-encryption-aws-kms-key-id"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if len(hi.Values("x-amz-website-redirect-location")) != 0 {
i.WebsiteRedirectLocation = thing_pointer(hi.Get("x-amz-website-redirect-location"))}
if len(hi.Values("x-amz-storage-class")) != 0 {
var s = hi.Get("x-amz-storage-class")
var x, err2 = intern_StorageClass(s)
if err2 != nil {input_errors["x-amz-storage-class"] = &Bb_input_error{"x-amz-storage-class", err2}} else {i.StorageClass = x}}
if len(hi.Values("x-amz-server-side-encryption")) != 0 {
var s = hi.Get("x-amz-server-side-encryption")
var x, err2 = intern_ServerSideEncryption(s)
if err2 != nil {input_errors["x-amz-server-side-encryption"] = &Bb_input_error{"x-amz-server-side-encryption", err2}} else {i.ServerSideEncryption = x}}
if len(hi.Values("x-amz-meta-")) != 0 {
var prefix = http.CanonicalHeaderKey("x-amz-meta-")
var bin map[string]string
for k, v := range hi {
if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}
i.Metadata = bin}
if len(hi.Values("x-amz-write-offset-bytes")) != 0 {
var s = hi.Get("x-amz-write-offset-bytes")
var x, err2 = strconv.ParseInt(s, 10, 64)
if err2 != nil {input_errors["x-amz-write-offset-bytes"] = &Bb_input_error{"x-amz-write-offset-bytes", err2}} else {i.WriteOffsetBytes = &x}}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
if len(hi.Values("x-amz-grant-write-acp")) != 0 {
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))}
if len(hi.Values("x-amz-grant-read-acp")) != 0 {
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))}
if len(hi.Values("x-amz-grant-read")) != 0 {
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))}
if len(hi.Values("x-amz-grant-full-control")) != 0 {
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))}
if len(hi.Values("If-None-Match")) != 0 {
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))}
if len(hi.Values("If-Match")) != 0 {
i.IfMatch = thing_pointer(hi.Get("If-Match"))}
if len(hi.Values("Expires")) != 0 {
var s = hi.Get("Expires")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["Expires"] = &Bb_input_error{"Expires", err2}} else {i.Expires = &x}}
if len(hi.Values("x-amz-checksum-sha256")) != 0 {
i.ChecksumSHA256 = thing_pointer(hi.Get("x-amz-checksum-sha256"))}
if len(hi.Values("x-amz-checksum-sha1")) != 0 {
i.ChecksumSHA1 = thing_pointer(hi.Get("x-amz-checksum-sha1"))}
if len(hi.Values("x-amz-checksum-crc64nvme")) != 0 {
i.ChecksumCRC64NVME = thing_pointer(hi.Get("x-amz-checksum-crc64nvme"))}
if len(hi.Values("x-amz-checksum-crc32c")) != 0 {
i.ChecksumCRC32C = thing_pointer(hi.Get("x-amz-checksum-crc32c"))}
if len(hi.Values("x-amz-checksum-crc32")) != 0 {
i.ChecksumCRC32 = thing_pointer(hi.Get("x-amz-checksum-crc32"))}
if len(hi.Values("x-amz-sdk-checksum-algorithm")) != 0 {
var s = hi.Get("x-amz-sdk-checksum-algorithm")
var x, err2 = intern_ChecksumAlgorithm(s)
if err2 != nil {input_errors["x-amz-sdk-checksum-algorithm"] = &Bb_input_error{"x-amz-sdk-checksum-algorithm", err2}} else {i.ChecksumAlgorithm = x}}
if len(hi.Values("Content-Type")) != 0 {
i.ContentType = thing_pointer(hi.Get("Content-Type"))}
if len(hi.Values("Content-MD5")) != 0 {
i.ContentMD5 = thing_pointer(hi.Get("Content-MD5"))}
if len(hi.Values("Content-Length")) != 0 {
var s = hi.Get("Content-Length")
var x, err2 = strconv.ParseInt(s, 10, 64)
if err2 != nil {input_errors["Content-Length"] = &Bb_input_error{"Content-Length", err2}} else {i.ContentLength = &x}}
if len(hi.Values("Content-Language")) != 0 {
i.ContentLanguage = thing_pointer(hi.Get("Content-Language"))}
if len(hi.Values("Content-Encoding")) != 0 {
i.ContentEncoding = thing_pointer(hi.Get("Content-Encoding"))}
if len(hi.Values("Content-Disposition")) != 0 {
i.ContentDisposition = thing_pointer(hi.Get("Content-Disposition"))}
if len(hi.Values("Cache-Control")) != 0 {
i.CacheControl = thing_pointer(hi.Get("Cache-Control"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
if len(hi.Values("x-amz-acl")) != 0 {
var s = hi.Get("x-amz-acl")
var x, err2 = intern_ObjectCannedACL(s)
if err2 != nil {input_errors["x-amz-acl"] = &Bb_input_error{"x-amz-acl", err2}} else {i.ACL = x}}
var o, err5 = bbs.PutObject(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_PutObjectResponse(*o)
if s.Expiration != nil {
ho.Add("x-amz-expiration", string(*s.Expiration))}
if s.ETag != nil {
ho.Add("ETag", string(*s.ETag))}
if s.ChecksumCRC32 != nil {
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))}
if s.ChecksumCRC32C != nil {
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))}
if s.ChecksumCRC64NVME != nil {
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))}
if s.ChecksumSHA1 != nil {
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))}
if s.ChecksumSHA256 != nil {
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))}
if s.ChecksumType != "" {
ho.Add("x-amz-checksum-type", string(s.ChecksumType))}
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
if s.SSECustomerAlgorithm != nil {
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))}
if s.SSECustomerKeyMD5 != nil {
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.SSEKMSEncryptionContext != nil {
ho.Add("x-amz-server-side-encryption-context", string(*s.SSEKMSEncryptionContext))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.Size != nil {
ho.Add("x-amz-object-size", strconv.FormatInt(*s.Size, 10))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_PutObjectTagging(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.PutObjectTaggingInput{}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-sdk-checksum-algorithm")) != 0 {
var s = hi.Get("x-amz-sdk-checksum-algorithm")
var x, err2 = intern_ChecksumAlgorithm(s)
if err2 != nil {input_errors["x-amz-sdk-checksum-algorithm"] = &Bb_input_error{"x-amz-sdk-checksum-algorithm", err2}} else {i.ChecksumAlgorithm = x}}
if len(hi.Values("Content-MD5")) != 0 {
i.ContentMD5 = thing_pointer(hi.Get("Content-MD5"))}
if qi.Has("versionId") {
i.VersionId = thing_pointer(qi.Get("versionId"))}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
{var x types.Tagging
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {panic(fmt.Errorf("No http body for types.Tagging: %w", err1))}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {panic(fmt.Errorf("Invalid http body for types.Tagging: %w", err2))}
i.Tagging = &x}
var o, err5 = bbs.PutObjectTagging(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_PutObjectTaggingResponse(*o)
if s.VersionId != nil {
ho.Add("x-amz-version-id", string(*s.VersionId))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_UploadPart(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.UploadPartInput{}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if qi.Has("uploadId") {
i.UploadId = thing_pointer(qi.Get("uploadId"))}
if qi.Has("partNumber") {
var s = qi.Get("partNumber")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["partNumber"] = &Bb_input_error{"partNumber", err2}} else {i.PartNumber = &x2}}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
if len(hi.Values("x-amz-checksum-sha256")) != 0 {
i.ChecksumSHA256 = thing_pointer(hi.Get("x-amz-checksum-sha256"))}
if len(hi.Values("x-amz-checksum-sha1")) != 0 {
i.ChecksumSHA1 = thing_pointer(hi.Get("x-amz-checksum-sha1"))}
if len(hi.Values("x-amz-checksum-crc64nvme")) != 0 {
i.ChecksumCRC64NVME = thing_pointer(hi.Get("x-amz-checksum-crc64nvme"))}
if len(hi.Values("x-amz-checksum-crc32c")) != 0 {
i.ChecksumCRC32C = thing_pointer(hi.Get("x-amz-checksum-crc32c"))}
if len(hi.Values("x-amz-checksum-crc32")) != 0 {
i.ChecksumCRC32 = thing_pointer(hi.Get("x-amz-checksum-crc32"))}
if len(hi.Values("x-amz-sdk-checksum-algorithm")) != 0 {
var s = hi.Get("x-amz-sdk-checksum-algorithm")
var x, err2 = intern_ChecksumAlgorithm(s)
if err2 != nil {input_errors["x-amz-sdk-checksum-algorithm"] = &Bb_input_error{"x-amz-sdk-checksum-algorithm", err2}} else {i.ChecksumAlgorithm = x}}
if len(hi.Values("Content-MD5")) != 0 {
i.ContentMD5 = thing_pointer(hi.Get("Content-MD5"))}
if len(hi.Values("Content-Length")) != 0 {
var s = hi.Get("Content-Length")
var x, err2 = strconv.ParseInt(s, 10, 64)
if err2 != nil {input_errors["Content-Length"] = &Bb_input_error{"Content-Length", err2}} else {i.ContentLength = &x}}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.UploadPart(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_UploadPartResponse(*o)
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
if s.ETag != nil {
ho.Add("ETag", string(*s.ETag))}
if s.ChecksumCRC32 != nil {
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))}
if s.ChecksumCRC32C != nil {
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))}
if s.ChecksumCRC64NVME != nil {
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))}
if s.ChecksumSHA1 != nil {
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))}
if s.ChecksumSHA256 != nil {
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))}
if s.SSECustomerAlgorithm != nil {
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))}
if s.SSECustomerKeyMD5 != nil {
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func h_UploadPartCopy(bbs *Bb_server, w http.ResponseWriter, r *http.Request) {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var input_errors = map[string]error{}
var ctx1 = r.Context()
var ctx2 = context.WithValue(ctx1, "request-id", bbs.make_request_id())
var ctx = context.WithValue(ctx2, "input-errors", input_errors)
var i = s3.UploadPartCopyInput{}
if len(hi.Values("x-amz-source-expected-bucket-owner")) != 0 {
i.ExpectedSourceBucketOwner = thing_pointer(hi.Get("x-amz-source-expected-bucket-owner"))}
if len(hi.Values("x-amz-expected-bucket-owner")) != 0 {
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))}
if len(hi.Values("x-amz-request-payer")) != 0 {
var s = hi.Get("x-amz-request-payer")
var x, err2 = intern_RequestPayer(s)
if err2 != nil {input_errors["x-amz-request-payer"] = &Bb_input_error{"x-amz-request-payer", err2}} else {i.RequestPayer = x}}
if len(hi.Values("x-amz-copy-source-server-side-encryption-customer-key-MD5")) != 0 {
i.CopySourceSSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-copy-source-server-side-encryption-customer-key")) != 0 {
i.CopySourceSSECustomerKey = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-copy-source-server-side-encryption-customer-algorithm")) != 0 {
i.CopySourceSSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-algorithm"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key-MD5")) != 0 {
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))}
if len(hi.Values("x-amz-server-side-encryption-customer-key")) != 0 {
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))}
if len(hi.Values("x-amz-server-side-encryption-customer-algorithm")) != 0 {
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))}
if qi.Has("uploadId") {
i.UploadId = thing_pointer(qi.Get("uploadId"))}
if qi.Has("partNumber") {
var s = qi.Get("partNumber")
var x1, err2 = strconv.ParseInt(s, 10, 32)
var x2 = int32(x1)
if err2 != nil {input_errors["partNumber"] = &Bb_input_error{"partNumber", err2}} else {i.PartNumber = &x2}}
{var x = r.PathValue("key")
if x != "" {i.Key = &x}}
if len(hi.Values("x-amz-copy-source-range")) != 0 {
i.CopySourceRange = thing_pointer(hi.Get("x-amz-copy-source-range"))}
if len(hi.Values("x-amz-copy-source-if-unmodified-since")) != 0 {
var s = hi.Get("x-amz-copy-source-if-unmodified-since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-copy-source-if-unmodified-since"] = &Bb_input_error{"x-amz-copy-source-if-unmodified-since", err2}} else {i.CopySourceIfUnmodifiedSince = &x}}
if len(hi.Values("x-amz-copy-source-if-none-match")) != 0 {
i.CopySourceIfNoneMatch = thing_pointer(hi.Get("x-amz-copy-source-if-none-match"))}
if len(hi.Values("x-amz-copy-source-if-modified-since")) != 0 {
var s = hi.Get("x-amz-copy-source-if-modified-since")
var x, err2 = time.Parse(time.RFC3339, s)
if err2 != nil {input_errors["x-amz-copy-source-if-modified-since"] = &Bb_input_error{"x-amz-copy-source-if-modified-since", err2}} else {i.CopySourceIfModifiedSince = &x}}
if len(hi.Values("x-amz-copy-source-if-match")) != 0 {
i.CopySourceIfMatch = thing_pointer(hi.Get("x-amz-copy-source-if-match"))}
if len(hi.Values("x-amz-copy-source")) != 0 {
i.CopySource = thing_pointer(hi.Get("x-amz-copy-source"))}
{var x = r.PathValue("bucket")
if x != "" {i.Bucket = &x}}
var o, err5 = bbs.UploadPartCopy(ctx, &i)
if err5 != nil {bbs.respond_on_action_error(ctx, w, r, err5); return}
var s = s_UploadPartCopyResponse(*o)
if s.CopySourceVersionId != nil {
ho.Add("x-amz-copy-source-version-id", string(*s.CopySourceVersionId))}
if s.ServerSideEncryption != "" {
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))}
if s.SSECustomerAlgorithm != nil {
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))}
if s.SSECustomerKeyMD5 != nil {
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))}
if s.SSEKMSKeyId != nil {
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))}
if s.BucketKeyEnabled != nil {
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))}
if s.RequestCharged != "" {
ho.Add("x-amz-request-charged", string(s.RequestCharged))}
ho.Set("Content-Type", "application/xml")
var co, err6 = xml.MarshalIndent(s, " ", "  ")
if err6 != nil {log.Fatal(err6)}
var status int = 200
w.WriteHeader(status)
var _, err7 = w.Write(co)
if err7 != nil {log.Fatal(err7)}
}
func intern_BucketCannedACL(s string) (types.BucketCannedACL, error) {
switch s {
case "private": return types.BucketCannedACLPrivate, nil
case "public-read": return types.BucketCannedACLPublicRead, nil
case "public-read-write": return types.BucketCannedACLPublicReadWrite, nil
case "authenticated-read": return types.BucketCannedACLAuthenticatedRead, nil
default: var err3 = &Bb_enum_intern_error{"types.BucketCannedACL", s}
return "_invalid_", err3}}
func intern_BucketLocationConstraint(s string) (types.BucketLocationConstraint, error) {
switch s {
case "af-south-1": return types.BucketLocationConstraintAfSouth1, nil
case "ap-east-1": return types.BucketLocationConstraintApEast1, nil
case "ap-northeast-1": return types.BucketLocationConstraintApNortheast1, nil
case "ap-northeast-2": return types.BucketLocationConstraintApNortheast2, nil
case "ap-northeast-3": return types.BucketLocationConstraintApNortheast3, nil
case "ap-south-1": return types.BucketLocationConstraintApSouth1, nil
case "ap-south-2": return types.BucketLocationConstraintApSouth2, nil
case "ap-southeast-1": return types.BucketLocationConstraintApSoutheast1, nil
case "ap-southeast-2": return types.BucketLocationConstraintApSoutheast2, nil
case "ap-southeast-3": return types.BucketLocationConstraintApSoutheast3, nil
case "ap-southeast-4": return types.BucketLocationConstraintApSoutheast4, nil
case "ap-southeast-5": return types.BucketLocationConstraintApSoutheast5, nil
case "ca-central-1": return types.BucketLocationConstraintCaCentral1, nil
case "cn-north-1": return types.BucketLocationConstraintCnNorth1, nil
case "cn-northwest-1": return types.BucketLocationConstraintCnNorthwest1, nil
case "EU": return types.BucketLocationConstraintEu, nil
case "eu-central-1": return types.BucketLocationConstraintEuCentral1, nil
case "eu-central-2": return types.BucketLocationConstraintEuCentral2, nil
case "eu-north-1": return types.BucketLocationConstraintEuNorth1, nil
case "eu-south-1": return types.BucketLocationConstraintEuSouth1, nil
case "eu-south-2": return types.BucketLocationConstraintEuSouth2, nil
case "eu-west-1": return types.BucketLocationConstraintEuWest1, nil
case "eu-west-2": return types.BucketLocationConstraintEuWest2, nil
case "eu-west-3": return types.BucketLocationConstraintEuWest3, nil
case "il-central-1": return types.BucketLocationConstraintIlCentral1, nil
case "me-central-1": return types.BucketLocationConstraintMeCentral1, nil
case "me-south-1": return types.BucketLocationConstraintMeSouth1, nil
case "sa-east-1": return types.BucketLocationConstraintSaEast1, nil
case "us-east-2": return types.BucketLocationConstraintUsEast2, nil
case "us-gov-east-1": return types.BucketLocationConstraintUsGovEast1, nil
case "us-gov-west-1": return types.BucketLocationConstraintUsGovWest1, nil
case "us-west-1": return types.BucketLocationConstraintUsWest1, nil
case "us-west-2": return types.BucketLocationConstraintUsWest2, nil
default: var err3 = &Bb_enum_intern_error{"types.BucketLocationConstraint", s}
return "_invalid_", err3}}
func intern_BucketType(s string) (types.BucketType, error) {
switch s {
case "Directory": return types.BucketTypeDirectory, nil
default: var err3 = &Bb_enum_intern_error{"types.BucketType", s}
return "_invalid_", err3}}
func intern_ChecksumAlgorithm(s string) (types.ChecksumAlgorithm, error) {
switch s {
case "CRC32": return types.ChecksumAlgorithmCrc32, nil
case "CRC32C": return types.ChecksumAlgorithmCrc32c, nil
case "SHA1": return types.ChecksumAlgorithmSha1, nil
case "SHA256": return types.ChecksumAlgorithmSha256, nil
case "CRC64NVME": return types.ChecksumAlgorithmCrc64nvme, nil
default: var err3 = &Bb_enum_intern_error{"types.ChecksumAlgorithm", s}
return "_invalid_", err3}}
func intern_ChecksumMode(s string) (types.ChecksumMode, error) {
switch s {
case "ENABLED": return types.ChecksumModeEnabled, nil
default: var err3 = &Bb_enum_intern_error{"types.ChecksumMode", s}
return "_invalid_", err3}}
func intern_ChecksumType(s string) (types.ChecksumType, error) {
switch s {
case "COMPOSITE": return types.ChecksumTypeComposite, nil
case "FULL_OBJECT": return types.ChecksumTypeFullObject, nil
default: var err3 = &Bb_enum_intern_error{"types.ChecksumType", s}
return "_invalid_", err3}}
func intern_DataRedundancy(s string) (types.DataRedundancy, error) {
switch s {
case "SingleAvailabilityZone": return types.DataRedundancySingleAvailabilityZone, nil
case "SingleLocalZone": return types.DataRedundancySingleLocalZone, nil
default: var err3 = &Bb_enum_intern_error{"types.DataRedundancy", s}
return "_invalid_", err3}}
func intern_EncodingType(s string) (types.EncodingType, error) {
switch s {
case "url": return types.EncodingTypeUrl, nil
default: var err3 = &Bb_enum_intern_error{"types.EncodingType", s}
return "_invalid_", err3}}
func intern_LocationType(s string) (types.LocationType, error) {
switch s {
case "AvailabilityZone": return types.LocationTypeAvailabilityZone, nil
case "LocalZone": return types.LocationTypeLocalZone, nil
default: var err3 = &Bb_enum_intern_error{"types.LocationType", s}
return "_invalid_", err3}}
func intern_MetadataDirective(s string) (types.MetadataDirective, error) {
switch s {
case "COPY": return types.MetadataDirectiveCopy, nil
case "REPLACE": return types.MetadataDirectiveReplace, nil
default: var err3 = &Bb_enum_intern_error{"types.MetadataDirective", s}
return "_invalid_", err3}}
func intern_ObjectAttributes(s string) (types.ObjectAttributes, error) {
switch s {
case "ETag": return types.ObjectAttributesEtag, nil
case "Checksum": return types.ObjectAttributesChecksum, nil
case "ObjectParts": return types.ObjectAttributesObjectParts, nil
case "StorageClass": return types.ObjectAttributesStorageClass, nil
case "ObjectSize": return types.ObjectAttributesObjectSize, nil
default: var err3 = &Bb_enum_intern_error{"types.ObjectAttributes", s}
return "_invalid_", err3}}
func intern_ObjectCannedACL(s string) (types.ObjectCannedACL, error) {
switch s {
case "private": return types.ObjectCannedACLPrivate, nil
case "public-read": return types.ObjectCannedACLPublicRead, nil
case "public-read-write": return types.ObjectCannedACLPublicReadWrite, nil
case "authenticated-read": return types.ObjectCannedACLAuthenticatedRead, nil
case "aws-exec-read": return types.ObjectCannedACLAwsExecRead, nil
case "bucket-owner-read": return types.ObjectCannedACLBucketOwnerRead, nil
case "bucket-owner-full-control": return types.ObjectCannedACLBucketOwnerFullControl, nil
default: var err3 = &Bb_enum_intern_error{"types.ObjectCannedACL", s}
return "_invalid_", err3}}
func intern_ObjectLockLegalHoldStatus(s string) (types.ObjectLockLegalHoldStatus, error) {
switch s {
case "ON": return types.ObjectLockLegalHoldStatusOn, nil
case "OFF": return types.ObjectLockLegalHoldStatusOff, nil
default: var err3 = &Bb_enum_intern_error{"types.ObjectLockLegalHoldStatus", s}
return "_invalid_", err3}}
func intern_ObjectLockMode(s string) (types.ObjectLockMode, error) {
switch s {
case "GOVERNANCE": return types.ObjectLockModeGovernance, nil
case "COMPLIANCE": return types.ObjectLockModeCompliance, nil
default: var err3 = &Bb_enum_intern_error{"types.ObjectLockMode", s}
return "_invalid_", err3}}
func intern_ObjectOwnership(s string) (types.ObjectOwnership, error) {
switch s {
case "BucketOwnerPreferred": return types.ObjectOwnershipBucketOwnerPreferred, nil
case "ObjectWriter": return types.ObjectOwnershipObjectWriter, nil
case "BucketOwnerEnforced": return types.ObjectOwnershipBucketOwnerEnforced, nil
default: var err3 = &Bb_enum_intern_error{"types.ObjectOwnership", s}
return "_invalid_", err3}}
func intern_OptionalObjectAttributes(s string) (types.OptionalObjectAttributes, error) {
switch s {
case "RestoreStatus": return types.OptionalObjectAttributesRestoreStatus, nil
default: var err3 = &Bb_enum_intern_error{"types.OptionalObjectAttributes", s}
return "_invalid_", err3}}
func intern_RequestPayer(s string) (types.RequestPayer, error) {
switch s {
case "requester": return types.RequestPayerRequester, nil
default: var err3 = &Bb_enum_intern_error{"types.RequestPayer", s}
return "_invalid_", err3}}
func intern_ServerSideEncryption(s string) (types.ServerSideEncryption, error) {
switch s {
case "AES256": return types.ServerSideEncryptionAes256, nil
case "aws:fsx": return types.ServerSideEncryptionAwsFsx, nil
case "aws:kms": return types.ServerSideEncryptionAwsKms, nil
case "aws:kms:dsse": return types.ServerSideEncryptionAwsKmsDsse, nil
default: var err3 = &Bb_enum_intern_error{"types.ServerSideEncryption", s}
return "_invalid_", err3}}
func intern_StorageClass(s string) (types.StorageClass, error) {
switch s {
case "STANDARD": return types.StorageClassStandard, nil
case "REDUCED_REDUNDANCY": return types.StorageClassReducedRedundancy, nil
case "STANDARD_IA": return types.StorageClassStandardIa, nil
case "ONEZONE_IA": return types.StorageClassOnezoneIa, nil
case "INTELLIGENT_TIERING": return types.StorageClassIntelligentTiering, nil
case "GLACIER": return types.StorageClassGlacier, nil
case "DEEP_ARCHIVE": return types.StorageClassDeepArchive, nil
case "OUTPOSTS": return types.StorageClassOutposts, nil
case "GLACIER_IR": return types.StorageClassGlacierIr, nil
case "SNOW": return types.StorageClassSnow, nil
case "EXPRESS_ONEZONE": return types.StorageClassExpressOnezone, nil
case "FSX_OPENZFS": return types.StorageClassFsxOpenzfs, nil
default: var err3 = &Bb_enum_intern_error{"types.StorageClass", s}
return "_invalid_", err3}}
func intern_TaggingDirective(s string) (types.TaggingDirective, error) {
switch s {
case "COPY": return types.TaggingDirectiveCopy, nil
case "REPLACE": return types.TaggingDirectiveReplace, nil
default: var err3 = &Bb_enum_intern_error{"types.TaggingDirective", s}
return "_invalid_", err3}}

