// handlers.go (2025-10-01)
// API-STUB.  Handler functions (h_XXXX) called from the
// dispatcher.
package server
import (
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
func h_AbortMultipartUpload(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.AbortMultipartUploadInput{}
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-if-match-initiated-time"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-if-match-initiated-time: %w", err2)}
i.IfMatchInitiatedTime = &x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.UploadId = thing_pointer(qi.Get("uploadId"))
var ctx = r.Context()
var o, err3 = bbs.AbortMultipartUpload(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_AbortMultipartUploadResponse(*o)
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 204
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_CompleteMultipartUpload(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.CompleteMultipartUploadInput{}
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))
i.IfMatch = thing_pointer(hi.Get("If-Match"))
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
{var x, err2 = strconv.ParseInt(hi.Get("x-amz-mp-object-size"), 10, 64)
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-mp-object-size: %w", err2)}
i.MpuObjectSize = &x}
{var x, err2 = import_ChecksumType(hi.Get("x-amz-checksum-type"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-checksum-type: %w", err2)}
i.ChecksumType = x}
i.ChecksumSHA256 = thing_pointer(hi.Get("x-amz-checksum-sha256"))
i.ChecksumSHA1 = thing_pointer(hi.Get("x-amz-checksum-sha1"))
i.ChecksumCRC64NVME = thing_pointer(hi.Get("x-amz-checksum-crc64nvme"))
i.ChecksumCRC32C = thing_pointer(hi.Get("x-amz-checksum-crc32c"))
i.ChecksumCRC32 = thing_pointer(hi.Get("x-amz-checksum-crc32"))
i.UploadId = thing_pointer(qi.Get("uploadId"))
{var x types.CompletedMultipartUpload
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {log.Fatal(err1); return err1}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {log.Fatal(err2); return err2}
i.MultipartUpload = &x}
var ctx = r.Context()
var o, err3 = bbs.CompleteMultipartUpload(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_CompleteMultipartUploadResponse(*o)
ho.Add("x-amz-expiration", string(*s.Expiration))
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_CopyObject(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.CopyObjectInput{}
i.ExpectedSourceBucketOwner = thing_pointer(hi.Get("x-amz-source-expected-bucket-owner"))
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_ObjectLockLegalHoldStatus(hi.Get("x-amz-object-lock-legal-hold"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-legal-hold: %w", err2)}
i.ObjectLockLegalHoldStatus = x}
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-object-lock-retain-until-date"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-retain-until-date: %w", err2)}
i.ObjectLockRetainUntilDate = &x}
{var x, err2 = import_ObjectLockMode(hi.Get("x-amz-object-lock-mode"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-mode: %w", err2)}
i.ObjectLockMode = x}
i.Tagging = thing_pointer(hi.Get("x-amz-tagging"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.CopySourceSSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key-MD5"))
i.CopySourceSSECustomerKey = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key"))
i.CopySourceSSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-algorithm"))
{var x, err2 = strconv.ParseBool(hi.Get("x-amz-server-side-encryption-bucket-key-enabled"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-server-side-encryption-bucket-key-enabled: %w", err2)}
i.BucketKeyEnabled = &x}
i.SSEKMSEncryptionContext = thing_pointer(hi.Get("x-amz-server-side-encryption-context"))
i.SSEKMSKeyId = thing_pointer(hi.Get("x-amz-server-side-encryption-aws-kms-key-id"))
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.WebsiteRedirectLocation = thing_pointer(hi.Get("x-amz-website-redirect-location"))
{var x, err2 = import_StorageClass(hi.Get("x-amz-storage-class"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-storage-class: %w", err2)}
i.StorageClass = x}
{var x, err2 = import_ServerSideEncryption(hi.Get("x-amz-server-side-encryption"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-server-side-encryption: %w", err2)}
i.ServerSideEncryption = x}
{var x, err2 = import_TaggingDirective(hi.Get("x-amz-tagging-directive"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-tagging-directive: %w", err2)}
i.TaggingDirective = x}
{var x, err2 = import_MetadataDirective(hi.Get("x-amz-metadata-directive"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-metadata-directive: %w", err2)}
i.MetadataDirective = x}
{var prefix = http.CanonicalHeaderKey("x-amz-meta-")
var bin map[string]string
for k, v := range hi {
if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}
i.Metadata = bin}
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("Expires"))
if err2 != nil {return fmt.Errorf("Bad parameter to Expires: %w", err2)}
i.Expires = &x}
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-copy-source-if-unmodified-since"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-copy-source-if-unmodified-since: %w", err2)}
i.CopySourceIfUnmodifiedSince = &x}
i.CopySourceIfNoneMatch = thing_pointer(hi.Get("x-amz-copy-source-if-none-match"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-copy-source-if-modified-since"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-copy-source-if-modified-since: %w", err2)}
i.CopySourceIfModifiedSince = &x}
i.CopySourceIfMatch = thing_pointer(hi.Get("x-amz-copy-source-if-match"))
i.CopySource = thing_pointer(hi.Get("x-amz-copy-source"))
i.ContentType = thing_pointer(hi.Get("Content-Type"))
i.ContentLanguage = thing_pointer(hi.Get("Content-Language"))
i.ContentEncoding = thing_pointer(hi.Get("Content-Encoding"))
i.ContentDisposition = thing_pointer(hi.Get("Content-Disposition"))
{var x, err2 = import_ChecksumAlgorithm(hi.Get("x-amz-checksum-algorithm"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-checksum-algorithm: %w", err2)}
i.ChecksumAlgorithm = x}
i.CacheControl = thing_pointer(hi.Get("Cache-Control"))
{var x, err2 = import_ObjectCannedACL(hi.Get("x-amz-acl"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-acl: %w", err2)}
i.ACL = x}
var ctx = r.Context()
var o, err3 = bbs.CopyObject(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_CopyObjectResponse(*o)
ho.Add("x-amz-expiration", string(*s.Expiration))
ho.Add("x-amz-copy-source-version-id", string(*s.CopySourceVersionId))
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-context", string(*s.SSEKMSEncryptionContext))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_CreateBucket(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.CreateBucketInput{}
{var x, err2 = import_ObjectOwnership(hi.Get("x-amz-object-ownership"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-ownership: %w", err2)}
i.ObjectOwnership = x}
{var x, err2 = strconv.ParseBool(hi.Get("x-amz-bucket-object-lock-enabled"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-bucket-object-lock-enabled: %w", err2)}
i.ObjectLockEnabledForBucket = &x}
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))
i.GrantWrite = thing_pointer(hi.Get("x-amz-grant-write"))
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))
{var x, err2 = import_BucketCannedACL(hi.Get("x-amz-acl"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-acl: %w", err2)}
i.ACL = x}
{var x types.CreateBucketConfiguration
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {log.Fatal(err1); return err1}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {log.Fatal(err2); return err2}
i.CreateBucketConfiguration = &x}
var ctx = r.Context()
var o, err3 = bbs.CreateBucket(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_CreateBucketResponse(*o)
ho.Add("Location", string(*s.Location))
ho.Add("x-amz-bucket-arn", string(*s.BucketArn))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_CreateMultipartUpload(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.CreateMultipartUploadInput{}
{var x, err2 = import_ChecksumType(hi.Get("x-amz-checksum-type"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-checksum-type: %w", err2)}
i.ChecksumType = x}
{var x, err2 = import_ChecksumAlgorithm(hi.Get("x-amz-checksum-algorithm"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-checksum-algorithm: %w", err2)}
i.ChecksumAlgorithm = x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_ObjectLockLegalHoldStatus(hi.Get("x-amz-object-lock-legal-hold"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-legal-hold: %w", err2)}
i.ObjectLockLegalHoldStatus = x}
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-object-lock-retain-until-date"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-retain-until-date: %w", err2)}
i.ObjectLockRetainUntilDate = &x}
{var x, err2 = import_ObjectLockMode(hi.Get("x-amz-object-lock-mode"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-mode: %w", err2)}
i.ObjectLockMode = x}
i.Tagging = thing_pointer(hi.Get("x-amz-tagging"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
{var x, err2 = strconv.ParseBool(hi.Get("x-amz-server-side-encryption-bucket-key-enabled"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-server-side-encryption-bucket-key-enabled: %w", err2)}
i.BucketKeyEnabled = &x}
i.SSEKMSEncryptionContext = thing_pointer(hi.Get("x-amz-server-side-encryption-context"))
i.SSEKMSKeyId = thing_pointer(hi.Get("x-amz-server-side-encryption-aws-kms-key-id"))
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.WebsiteRedirectLocation = thing_pointer(hi.Get("x-amz-website-redirect-location"))
{var x, err2 = import_StorageClass(hi.Get("x-amz-storage-class"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-storage-class: %w", err2)}
i.StorageClass = x}
{var x, err2 = import_ServerSideEncryption(hi.Get("x-amz-server-side-encryption"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-server-side-encryption: %w", err2)}
i.ServerSideEncryption = x}
{var prefix = http.CanonicalHeaderKey("x-amz-meta-")
var bin map[string]string
for k, v := range hi {
if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}
i.Metadata = bin}
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("Expires"))
if err2 != nil {return fmt.Errorf("Bad parameter to Expires: %w", err2)}
i.Expires = &x}
i.ContentType = thing_pointer(hi.Get("Content-Type"))
i.ContentLanguage = thing_pointer(hi.Get("Content-Language"))
i.ContentEncoding = thing_pointer(hi.Get("Content-Encoding"))
i.ContentDisposition = thing_pointer(hi.Get("Content-Disposition"))
i.CacheControl = thing_pointer(hi.Get("Cache-Control"))
{var x, err2 = import_ObjectCannedACL(hi.Get("x-amz-acl"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-acl: %w", err2)}
i.ACL = x}
var ctx = r.Context()
var o, err3 = bbs.CreateMultipartUpload(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_CreateMultipartUploadResponse(*o)
ho.Add("x-amz-abort-date", s.AbortDate.String())
ho.Add("x-amz-abort-rule-id", string(*s.AbortRuleId))
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-context", string(*s.SSEKMSEncryptionContext))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Add("x-amz-checksum-algorithm", string(s.ChecksumAlgorithm))
ho.Add("x-amz-checksum-type", string(s.ChecksumType))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_DeleteBucket(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.DeleteBucketInput{}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
var ctx = r.Context()
var _, err3 = bbs.DeleteBucket(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
ho.Set("Content-Type", "application/xml")
var status int = 204
w.WriteHeader(status)
return nil}
func h_DeleteObject(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.DeleteObjectInput{}
{var x, err2 = strconv.ParseInt(hi.Get("x-amz-if-match-size"), 10, 64)
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-if-match-size: %w", err2)}
i.IfMatchSize = &x}
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-if-match-last-modified-time"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-if-match-last-modified-time: %w", err2)}
i.IfMatchLastModifiedTime = &x}
i.IfMatch = thing_pointer(hi.Get("If-Match"))
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = strconv.ParseBool(hi.Get("x-amz-bypass-governance-retention"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-bypass-governance-retention: %w", err2)}
i.BypassGovernanceRetention = &x}
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.VersionId = thing_pointer(qi.Get("versionId"))
i.MFA = thing_pointer(hi.Get("x-amz-mfa"))
var ctx = r.Context()
var o, err3 = bbs.DeleteObject(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_DeleteObjectResponse(*o)
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 204
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_DeleteObjects(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.DeleteObjectsInput{}
{var x, err2 = import_ChecksumAlgorithm(hi.Get("x-amz-sdk-checksum-algorithm"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-sdk-checksum-algorithm: %w", err2)}
i.ChecksumAlgorithm = x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = strconv.ParseBool(hi.Get("x-amz-bypass-governance-retention"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-bypass-governance-retention: %w", err2)}
i.BypassGovernanceRetention = &x}
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.MFA = thing_pointer(hi.Get("x-amz-mfa"))
{var x types.Delete
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {log.Fatal(err1); return err1}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {log.Fatal(err2); return err2}
i.Delete = &x}
var ctx = r.Context()
var o, err3 = bbs.DeleteObjects(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_DeleteObjectsResponse(*o)
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_DeleteObjectTagging(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.DeleteObjectTaggingInput{}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
i.VersionId = thing_pointer(qi.Get("versionId"))
var ctx = r.Context()
var o, err3 = bbs.DeleteObjectTagging(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_DeleteObjectTaggingResponse(*o)
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 204
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_GetObject(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.GetObjectInput{}
{var x, err2 = import_ChecksumMode(hi.Get("x-amz-checksum-mode"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-checksum-mode: %w", err2)}
i.ChecksumMode = x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x1, err2 = strconv.ParseInt(qi.Get("partNumber"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to partNumber: %w", err2)}
var x2 = int32(x1)
i.PartNumber = &x2}
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.VersionId = thing_pointer(qi.Get("versionId"))
{var x, err2 = time.Parse(time.RFC3339, qi.Get("response-expires"))
if err2 != nil {return fmt.Errorf("Bad parameter to response-expires: %w", err2)}
i.ResponseExpires = &x}
i.ResponseContentType = thing_pointer(qi.Get("response-content-type"))
i.ResponseContentLanguage = thing_pointer(qi.Get("response-content-language"))
i.ResponseContentEncoding = thing_pointer(qi.Get("response-content-encoding"))
i.ResponseContentDisposition = thing_pointer(qi.Get("response-content-disposition"))
i.ResponseCacheControl = thing_pointer(qi.Get("response-cache-control"))
i.Range = thing_pointer(hi.Get("Range"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("If-Unmodified-Since"))
if err2 != nil {return fmt.Errorf("Bad parameter to If-Unmodified-Since: %w", err2)}
i.IfUnmodifiedSince = &x}
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("If-Modified-Since"))
if err2 != nil {return fmt.Errorf("Bad parameter to If-Modified-Since: %w", err2)}
i.IfModifiedSince = &x}
i.IfMatch = thing_pointer(hi.Get("If-Match"))
var ctx = r.Context()
var o, err3 = bbs.GetObject(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_GetObjectResponse(*o)
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))
ho.Add("accept-ranges", string(*s.AcceptRanges))
ho.Add("x-amz-expiration", string(*s.Expiration))
ho.Add("x-amz-restore", string(*s.Restore))
ho.Add("Last-Modified", s.LastModified.String())
ho.Add("Content-Length", strconv.FormatInt(*s.ContentLength, 10))
ho.Add("ETag", string(*s.ETag))
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))
ho.Add("x-amz-checksum-type", string(s.ChecksumType))
ho.Add("x-amz-missing-meta", strconv.FormatInt(int64(*s.MissingMeta), 10))
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Add("Cache-Control", string(*s.CacheControl))
ho.Add("Content-Disposition", string(*s.ContentDisposition))
ho.Add("Content-Encoding", string(*s.ContentEncoding))
ho.Add("Content-Language", string(*s.ContentLanguage))
ho.Add("Content-Range", string(*s.ContentRange))
ho.Add("Content-Type", string(*s.ContentType))
ho.Add("Expires", s.Expires.String())
ho.Add("x-amz-website-redirect-location", string(*s.WebsiteRedirectLocation))
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-storage-class", string(s.StorageClass))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Add("x-amz-replication-status", string(s.ReplicationStatus))
ho.Add("x-amz-mp-parts-count", strconv.FormatInt(int64(*s.PartsCount), 10))
ho.Add("x-amz-tagging-count", strconv.FormatInt(int64(*s.TagCount), 10))
ho.Add("x-amz-object-lock-mode", string(s.ObjectLockMode))
ho.Add("x-amz-object-lock-retain-until-date", s.ObjectLockRetainUntilDate.String())
ho.Add("x-amz-object-lock-legal-hold", string(s.ObjectLockLegalHoldStatus))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_GetObjectAttributes(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.GetObjectAttributesInput{}
{var rhs = hi.Values("x-amz-object-attributes")
var bin []types.ObjectAttributes
for _, v := range slices.All(rhs) {
{var x, err2 = import_ObjectAttributes(v)
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-attributes: %w", err2)}
bin = append(bin, x)}
}
i.ObjectAttributes = bin}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.PartNumberMarker = thing_pointer(hi.Get("x-amz-part-number-marker"))
{var x1, err2 = strconv.ParseInt(hi.Get("x-amz-max-parts"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-max-parts: %w", err2)}
var x2 = int32(x1)
i.MaxParts = &x2}
i.VersionId = thing_pointer(qi.Get("versionId"))
var ctx = r.Context()
var o, err3 = bbs.GetObjectAttributes(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_GetObjectAttributesResponse(*o)
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))
ho.Add("Last-Modified", s.LastModified.String())
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_GetObjectTagging(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.GetObjectTaggingInput{}
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
i.VersionId = thing_pointer(qi.Get("versionId"))
var ctx = r.Context()
var o, err3 = bbs.GetObjectTagging(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_GetObjectTaggingResponse(*o)
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_HeadBucket(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.HeadBucketInput{}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
var ctx = r.Context()
var o, err3 = bbs.HeadBucket(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_HeadBucketResponse(*o)
ho.Add("x-amz-bucket-arn", string(*s.BucketArn))
ho.Add("x-amz-bucket-location-type", string(s.BucketLocationType))
ho.Add("x-amz-bucket-location-name", string(*s.BucketLocationName))
ho.Add("x-amz-bucket-region", string(*s.BucketRegion))
ho.Add("x-amz-access-point-alias", strconv.FormatBool(*s.AccessPointAlias))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_HeadObject(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.HeadObjectInput{}
{var x, err2 = import_ChecksumMode(hi.Get("x-amz-checksum-mode"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-checksum-mode: %w", err2)}
i.ChecksumMode = x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x1, err2 = strconv.ParseInt(qi.Get("partNumber"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to partNumber: %w", err2)}
var x2 = int32(x1)
i.PartNumber = &x2}
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.VersionId = thing_pointer(qi.Get("versionId"))
{var x, err2 = time.Parse(time.RFC3339, qi.Get("response-expires"))
if err2 != nil {return fmt.Errorf("Bad parameter to response-expires: %w", err2)}
i.ResponseExpires = &x}
i.ResponseContentType = thing_pointer(qi.Get("response-content-type"))
i.ResponseContentLanguage = thing_pointer(qi.Get("response-content-language"))
i.ResponseContentEncoding = thing_pointer(qi.Get("response-content-encoding"))
i.ResponseContentDisposition = thing_pointer(qi.Get("response-content-disposition"))
i.ResponseCacheControl = thing_pointer(qi.Get("response-cache-control"))
i.Range = thing_pointer(hi.Get("Range"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("If-Unmodified-Since"))
if err2 != nil {return fmt.Errorf("Bad parameter to If-Unmodified-Since: %w", err2)}
i.IfUnmodifiedSince = &x}
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("If-Modified-Since"))
if err2 != nil {return fmt.Errorf("Bad parameter to If-Modified-Since: %w", err2)}
i.IfModifiedSince = &x}
i.IfMatch = thing_pointer(hi.Get("If-Match"))
var ctx = r.Context()
var o, err3 = bbs.HeadObject(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_HeadObjectResponse(*o)
ho.Add("x-amz-delete-marker", strconv.FormatBool(*s.DeleteMarker))
ho.Add("accept-ranges", string(*s.AcceptRanges))
ho.Add("x-amz-expiration", string(*s.Expiration))
ho.Add("x-amz-restore", string(*s.Restore))
ho.Add("x-amz-archive-status", string(s.ArchiveStatus))
ho.Add("Last-Modified", s.LastModified.String())
ho.Add("Content-Length", strconv.FormatInt(*s.ContentLength, 10))
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))
ho.Add("x-amz-checksum-type", string(s.ChecksumType))
ho.Add("ETag", string(*s.ETag))
ho.Add("x-amz-missing-meta", strconv.FormatInt(int64(*s.MissingMeta), 10))
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Add("Cache-Control", string(*s.CacheControl))
ho.Add("Content-Disposition", string(*s.ContentDisposition))
ho.Add("Content-Encoding", string(*s.ContentEncoding))
ho.Add("Content-Language", string(*s.ContentLanguage))
ho.Add("Content-Type", string(*s.ContentType))
ho.Add("Content-Range", string(*s.ContentRange))
ho.Add("Expires", s.Expires.String())
ho.Add("x-amz-website-redirect-location", string(*s.WebsiteRedirectLocation))
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-storage-class", string(s.StorageClass))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Add("x-amz-replication-status", string(s.ReplicationStatus))
ho.Add("x-amz-mp-parts-count", strconv.FormatInt(int64(*s.PartsCount), 10))
ho.Add("x-amz-tagging-count", strconv.FormatInt(int64(*s.TagCount), 10))
ho.Add("x-amz-object-lock-mode", string(s.ObjectLockMode))
ho.Add("x-amz-object-lock-retain-until-date", s.ObjectLockRetainUntilDate.String())
ho.Add("x-amz-object-lock-legal-hold", string(s.ObjectLockLegalHoldStatus))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_ListBuckets(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.ListBucketsInput{}
i.BucketRegion = thing_pointer(qi.Get("bucket-region"))
i.Prefix = thing_pointer(qi.Get("prefix"))
i.ContinuationToken = thing_pointer(qi.Get("continuation-token"))
{var x1, err2 = strconv.ParseInt(qi.Get("max-buckets"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to max-buckets: %w", err2)}
var x2 = int32(x1)
i.MaxBuckets = &x2}
var ctx = r.Context()
var o, err3 = bbs.ListBuckets(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_ListBucketsResponse(*o)
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_ListMultipartUploads(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.ListMultipartUploadsInput{}
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
i.UploadIdMarker = thing_pointer(qi.Get("upload-id-marker"))
i.Prefix = thing_pointer(qi.Get("prefix"))
{var x1, err2 = strconv.ParseInt(qi.Get("max-uploads"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to max-uploads: %w", err2)}
var x2 = int32(x1)
i.MaxUploads = &x2}
i.KeyMarker = thing_pointer(qi.Get("key-marker"))
{var x, err2 = import_EncodingType(qi.Get("encoding-type"))
if err2 != nil {return fmt.Errorf("Bad parameter to encoding-type: %w", err2)}
i.EncodingType = x}
i.Delimiter = thing_pointer(qi.Get("delimiter"))
var ctx = r.Context()
var o, err3 = bbs.ListMultipartUploads(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_ListMultipartUploadsResponse(*o)
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_ListObjects(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.ListObjectsInput{}
{var rhs = hi.Values("x-amz-optional-object-attributes")
var bin []types.OptionalObjectAttributes
for _, v := range slices.All(rhs) {
{var x, err2 = import_OptionalObjectAttributes(v)
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-optional-object-attributes: %w", err2)}
bin = append(bin, x)}
}
i.OptionalObjectAttributes = bin}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.Prefix = thing_pointer(qi.Get("prefix"))
{var x1, err2 = strconv.ParseInt(qi.Get("max-keys"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to max-keys: %w", err2)}
var x2 = int32(x1)
i.MaxKeys = &x2}
i.Marker = thing_pointer(qi.Get("marker"))
{var x, err2 = import_EncodingType(qi.Get("encoding-type"))
if err2 != nil {return fmt.Errorf("Bad parameter to encoding-type: %w", err2)}
i.EncodingType = x}
i.Delimiter = thing_pointer(qi.Get("delimiter"))
var ctx = r.Context()
var o, err3 = bbs.ListObjects(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_ListObjectsResponse(*o)
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_ListObjectsV2(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.ListObjectsV2Input{}
{var rhs = hi.Values("x-amz-optional-object-attributes")
var bin []types.OptionalObjectAttributes
for _, v := range slices.All(rhs) {
{var x, err2 = import_OptionalObjectAttributes(v)
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-optional-object-attributes: %w", err2)}
bin = append(bin, x)}
}
i.OptionalObjectAttributes = bin}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.StartAfter = thing_pointer(qi.Get("start-after"))
{var x, err2 = strconv.ParseBool(qi.Get("fetch-owner"))
if err2 != nil {return fmt.Errorf("Bad parameter to fetch-owner: %w", err2)}
i.FetchOwner = &x}
i.ContinuationToken = thing_pointer(qi.Get("continuation-token"))
i.Prefix = thing_pointer(qi.Get("prefix"))
{var x1, err2 = strconv.ParseInt(qi.Get("max-keys"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to max-keys: %w", err2)}
var x2 = int32(x1)
i.MaxKeys = &x2}
{var x, err2 = import_EncodingType(qi.Get("encoding-type"))
if err2 != nil {return fmt.Errorf("Bad parameter to encoding-type: %w", err2)}
i.EncodingType = x}
i.Delimiter = thing_pointer(qi.Get("delimiter"))
var ctx = r.Context()
var o, err3 = bbs.ListObjectsV2(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_ListObjectsV2Response(*o)
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_ListParts(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.ListPartsInput{}
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.UploadId = thing_pointer(qi.Get("uploadId"))
i.PartNumberMarker = thing_pointer(qi.Get("part-number-marker"))
{var x1, err2 = strconv.ParseInt(qi.Get("max-parts"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to max-parts: %w", err2)}
var x2 = int32(x1)
i.MaxParts = &x2}
var ctx = r.Context()
var o, err3 = bbs.ListParts(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_ListPartsResponse(*o)
ho.Add("x-amz-abort-date", s.AbortDate.String())
ho.Add("x-amz-abort-rule-id", string(*s.AbortRuleId))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_PutObject(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.PutObjectInput{}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_ObjectLockLegalHoldStatus(hi.Get("x-amz-object-lock-legal-hold"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-legal-hold: %w", err2)}
i.ObjectLockLegalHoldStatus = x}
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-object-lock-retain-until-date"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-retain-until-date: %w", err2)}
i.ObjectLockRetainUntilDate = &x}
{var x, err2 = import_ObjectLockMode(hi.Get("x-amz-object-lock-mode"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-object-lock-mode: %w", err2)}
i.ObjectLockMode = x}
i.Tagging = thing_pointer(hi.Get("x-amz-tagging"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
{var x, err2 = strconv.ParseBool(hi.Get("x-amz-server-side-encryption-bucket-key-enabled"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-server-side-encryption-bucket-key-enabled: %w", err2)}
i.BucketKeyEnabled = &x}
i.SSEKMSEncryptionContext = thing_pointer(hi.Get("x-amz-server-side-encryption-context"))
i.SSEKMSKeyId = thing_pointer(hi.Get("x-amz-server-side-encryption-aws-kms-key-id"))
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.WebsiteRedirectLocation = thing_pointer(hi.Get("x-amz-website-redirect-location"))
{var x, err2 = import_StorageClass(hi.Get("x-amz-storage-class"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-storage-class: %w", err2)}
i.StorageClass = x}
{var x, err2 = import_ServerSideEncryption(hi.Get("x-amz-server-side-encryption"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-server-side-encryption: %w", err2)}
i.ServerSideEncryption = x}
{var prefix = http.CanonicalHeaderKey("x-amz-meta-")
var bin map[string]string
for k, v := range hi {
if strings.HasPrefix(k, prefix) {bin[k] = v[0]}}
i.Metadata = bin}
{var x, err2 = strconv.ParseInt(hi.Get("x-amz-write-offset-bytes"), 10, 64)
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-write-offset-bytes: %w", err2)}
i.WriteOffsetBytes = &x}
i.GrantWriteACP = thing_pointer(hi.Get("x-amz-grant-write-acp"))
i.GrantReadACP = thing_pointer(hi.Get("x-amz-grant-read-acp"))
i.GrantRead = thing_pointer(hi.Get("x-amz-grant-read"))
i.GrantFullControl = thing_pointer(hi.Get("x-amz-grant-full-control"))
i.IfNoneMatch = thing_pointer(hi.Get("If-None-Match"))
i.IfMatch = thing_pointer(hi.Get("If-Match"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("Expires"))
if err2 != nil {return fmt.Errorf("Bad parameter to Expires: %w", err2)}
i.Expires = &x}
i.ChecksumSHA256 = thing_pointer(hi.Get("x-amz-checksum-sha256"))
i.ChecksumSHA1 = thing_pointer(hi.Get("x-amz-checksum-sha1"))
i.ChecksumCRC64NVME = thing_pointer(hi.Get("x-amz-checksum-crc64nvme"))
i.ChecksumCRC32C = thing_pointer(hi.Get("x-amz-checksum-crc32c"))
i.ChecksumCRC32 = thing_pointer(hi.Get("x-amz-checksum-crc32"))
{var x, err2 = import_ChecksumAlgorithm(hi.Get("x-amz-sdk-checksum-algorithm"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-sdk-checksum-algorithm: %w", err2)}
i.ChecksumAlgorithm = x}
i.ContentType = thing_pointer(hi.Get("Content-Type"))
i.ContentMD5 = thing_pointer(hi.Get("Content-MD5"))
{var x, err2 = strconv.ParseInt(hi.Get("Content-Length"), 10, 64)
if err2 != nil {return fmt.Errorf("Bad parameter to Content-Length: %w", err2)}
i.ContentLength = &x}
i.ContentLanguage = thing_pointer(hi.Get("Content-Language"))
i.ContentEncoding = thing_pointer(hi.Get("Content-Encoding"))
i.ContentDisposition = thing_pointer(hi.Get("Content-Disposition"))
i.CacheControl = thing_pointer(hi.Get("Cache-Control"))
{var x, err2 = import_ObjectCannedACL(hi.Get("x-amz-acl"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-acl: %w", err2)}
i.ACL = x}
var ctx = r.Context()
var o, err3 = bbs.PutObject(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_PutObjectResponse(*o)
ho.Add("x-amz-expiration", string(*s.Expiration))
ho.Add("ETag", string(*s.ETag))
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))
ho.Add("x-amz-checksum-type", string(s.ChecksumType))
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-context", string(*s.SSEKMSEncryptionContext))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-object-size", strconv.FormatInt(*s.Size, 10))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_PutObjectTagging(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.PutObjectTaggingInput{}
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_ChecksumAlgorithm(hi.Get("x-amz-sdk-checksum-algorithm"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-sdk-checksum-algorithm: %w", err2)}
i.ChecksumAlgorithm = x}
i.ContentMD5 = thing_pointer(hi.Get("Content-MD5"))
i.VersionId = thing_pointer(qi.Get("versionId"))
{var x types.Tagging
var bs, err1 = io.ReadAll(r.Body)
if err1 != nil {log.Fatal(err1); return err1}
var err2 = xml.Unmarshal(bs, &x)
if err2 != nil {log.Fatal(err2); return err2}
i.Tagging = &x}
var ctx = r.Context()
var o, err3 = bbs.PutObjectTagging(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_PutObjectTaggingResponse(*o)
ho.Add("x-amz-version-id", string(*s.VersionId))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_UploadPart(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.UploadPartInput{}
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.UploadId = thing_pointer(qi.Get("uploadId"))
{var x1, err2 = strconv.ParseInt(qi.Get("partNumber"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to partNumber: %w", err2)}
var x2 = int32(x1)
i.PartNumber = &x2}
i.ChecksumSHA256 = thing_pointer(hi.Get("x-amz-checksum-sha256"))
i.ChecksumSHA1 = thing_pointer(hi.Get("x-amz-checksum-sha1"))
i.ChecksumCRC64NVME = thing_pointer(hi.Get("x-amz-checksum-crc64nvme"))
i.ChecksumCRC32C = thing_pointer(hi.Get("x-amz-checksum-crc32c"))
i.ChecksumCRC32 = thing_pointer(hi.Get("x-amz-checksum-crc32"))
{var x, err2 = import_ChecksumAlgorithm(hi.Get("x-amz-sdk-checksum-algorithm"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-sdk-checksum-algorithm: %w", err2)}
i.ChecksumAlgorithm = x}
i.ContentMD5 = thing_pointer(hi.Get("Content-MD5"))
{var x, err2 = strconv.ParseInt(hi.Get("Content-Length"), 10, 64)
if err2 != nil {return fmt.Errorf("Bad parameter to Content-Length: %w", err2)}
i.ContentLength = &x}
var ctx = r.Context()
var o, err3 = bbs.UploadPart(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_UploadPartResponse(*o)
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("ETag", string(*s.ETag))
ho.Add("x-amz-checksum-crc32", string(*s.ChecksumCRC32))
ho.Add("x-amz-checksum-crc32c", string(*s.ChecksumCRC32C))
ho.Add("x-amz-checksum-crc64nvme", string(*s.ChecksumCRC64NVME))
ho.Add("x-amz-checksum-sha1", string(*s.ChecksumSHA1))
ho.Add("x-amz-checksum-sha256", string(*s.ChecksumSHA256))
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func h_UploadPartCopy(bbs *BB_server, w http.ResponseWriter, r *http.Request) error {
var qi = r.URL.Query()
var hi = r.Header
var ho = w.Header()
// Mark variables used to avoid unused errors:
var _, _, _ = qi, hi, ho
var i = s3.UploadPartCopyInput{}
i.ExpectedSourceBucketOwner = thing_pointer(hi.Get("x-amz-source-expected-bucket-owner"))
i.ExpectedBucketOwner = thing_pointer(hi.Get("x-amz-expected-bucket-owner"))
{var x, err2 = import_RequestPayer(hi.Get("x-amz-request-payer"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-request-payer: %w", err2)}
i.RequestPayer = x}
i.CopySourceSSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key-MD5"))
i.CopySourceSSECustomerKey = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-key"))
i.CopySourceSSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-copy-source-server-side-encryption-customer-algorithm"))
i.SSECustomerKeyMD5 = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key-MD5"))
i.SSECustomerKey = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-key"))
i.SSECustomerAlgorithm = thing_pointer(hi.Get("x-amz-server-side-encryption-customer-algorithm"))
i.UploadId = thing_pointer(qi.Get("uploadId"))
{var x1, err2 = strconv.ParseInt(qi.Get("partNumber"), 10, 32)
if err2 != nil {return fmt.Errorf("Bad parameter to partNumber: %w", err2)}
var x2 = int32(x1)
i.PartNumber = &x2}
i.CopySourceRange = thing_pointer(hi.Get("x-amz-copy-source-range"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-copy-source-if-unmodified-since"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-copy-source-if-unmodified-since: %w", err2)}
i.CopySourceIfUnmodifiedSince = &x}
i.CopySourceIfNoneMatch = thing_pointer(hi.Get("x-amz-copy-source-if-none-match"))
{var x, err2 = time.Parse(time.RFC3339, hi.Get("x-amz-copy-source-if-modified-since"))
if err2 != nil {return fmt.Errorf("Bad parameter to x-amz-copy-source-if-modified-since: %w", err2)}
i.CopySourceIfModifiedSince = &x}
i.CopySourceIfMatch = thing_pointer(hi.Get("x-amz-copy-source-if-match"))
i.CopySource = thing_pointer(hi.Get("x-amz-copy-source"))
var ctx = r.Context()
var o, err3 = bbs.UploadPartCopy(ctx, &i)
if err3 != nil {log.Fatal(err3); return err3}
var s = s_UploadPartCopyResponse(*o)
ho.Add("x-amz-copy-source-version-id", string(*s.CopySourceVersionId))
ho.Add("x-amz-server-side-encryption", string(s.ServerSideEncryption))
ho.Add("x-amz-server-side-encryption-customer-algorithm", string(*s.SSECustomerAlgorithm))
ho.Add("x-amz-server-side-encryption-customer-key-MD5", string(*s.SSECustomerKeyMD5))
ho.Add("x-amz-server-side-encryption-aws-kms-key-id", string(*s.SSEKMSKeyId))
ho.Add("x-amz-server-side-encryption-bucket-key-enabled", strconv.FormatBool(*s.BucketKeyEnabled))
ho.Add("x-amz-request-charged", string(s.RequestCharged))
ho.Set("Content-Type", "application/xml")
var co, err5 = xml.MarshalIndent(s, " ", "  ")
if err5 != nil {log.Fatal(err5); return err5}
var status int = 200
w.WriteHeader(status)
var _, err6 = w.Write(co)
if err6 != nil {log.Fatal(err6); return err6}
return nil}
func import_BucketCannedACL(s string) (types.BucketCannedACL, error) {
switch s {
case "private": return types.BucketCannedACLPrivate, nil
case "public-read": return types.BucketCannedACLPublicRead, nil
case "public-read-write": return types.BucketCannedACLPublicReadWrite, nil
case "authenticated-read": return types.BucketCannedACLAuthenticatedRead, nil
default: var err1 = fmt.Errorf("interning an enum (types.BucketCannedACL) %#v", s)
log.Print(err1); return "", err1}}
func import_BucketLocationConstraint(s string) (types.BucketLocationConstraint, error) {
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
default: var err1 = fmt.Errorf("interning an enum (types.BucketLocationConstraint) %#v", s)
log.Print(err1); return "", err1}}
func import_BucketType(s string) (types.BucketType, error) {
switch s {
case "Directory": return types.BucketTypeDirectory, nil
default: var err1 = fmt.Errorf("interning an enum (types.BucketType) %#v", s)
log.Print(err1); return "", err1}}
func import_ChecksumAlgorithm(s string) (types.ChecksumAlgorithm, error) {
switch s {
case "CRC32": return types.ChecksumAlgorithmCrc32, nil
case "CRC32C": return types.ChecksumAlgorithmCrc32c, nil
case "SHA1": return types.ChecksumAlgorithmSha1, nil
case "SHA256": return types.ChecksumAlgorithmSha256, nil
case "CRC64NVME": return types.ChecksumAlgorithmCrc64nvme, nil
default: var err1 = fmt.Errorf("interning an enum (types.ChecksumAlgorithm) %#v", s)
log.Print(err1); return "", err1}}
func import_ChecksumMode(s string) (types.ChecksumMode, error) {
switch s {
case "ENABLED": return types.ChecksumModeEnabled, nil
default: var err1 = fmt.Errorf("interning an enum (types.ChecksumMode) %#v", s)
log.Print(err1); return "", err1}}
func import_ChecksumType(s string) (types.ChecksumType, error) {
switch s {
case "COMPOSITE": return types.ChecksumTypeComposite, nil
case "FULL_OBJECT": return types.ChecksumTypeFullObject, nil
default: var err1 = fmt.Errorf("interning an enum (types.ChecksumType) %#v", s)
log.Print(err1); return "", err1}}
func import_DataRedundancy(s string) (types.DataRedundancy, error) {
switch s {
case "SingleAvailabilityZone": return types.DataRedundancySingleAvailabilityZone, nil
case "SingleLocalZone": return types.DataRedundancySingleLocalZone, nil
default: var err1 = fmt.Errorf("interning an enum (types.DataRedundancy) %#v", s)
log.Print(err1); return "", err1}}
func import_EncodingType(s string) (types.EncodingType, error) {
switch s {
case "url": return types.EncodingTypeUrl, nil
default: var err1 = fmt.Errorf("interning an enum (types.EncodingType) %#v", s)
log.Print(err1); return "", err1}}
func import_LocationType(s string) (types.LocationType, error) {
switch s {
case "AvailabilityZone": return types.LocationTypeAvailabilityZone, nil
case "LocalZone": return types.LocationTypeLocalZone, nil
default: var err1 = fmt.Errorf("interning an enum (types.LocationType) %#v", s)
log.Print(err1); return "", err1}}
func import_MetadataDirective(s string) (types.MetadataDirective, error) {
switch s {
case "COPY": return types.MetadataDirectiveCopy, nil
case "REPLACE": return types.MetadataDirectiveReplace, nil
default: var err1 = fmt.Errorf("interning an enum (types.MetadataDirective) %#v", s)
log.Print(err1); return "", err1}}
func import_ObjectAttributes(s string) (types.ObjectAttributes, error) {
switch s {
case "ETag": return types.ObjectAttributesEtag, nil
case "Checksum": return types.ObjectAttributesChecksum, nil
case "ObjectParts": return types.ObjectAttributesObjectParts, nil
case "StorageClass": return types.ObjectAttributesStorageClass, nil
case "ObjectSize": return types.ObjectAttributesObjectSize, nil
default: var err1 = fmt.Errorf("interning an enum (types.ObjectAttributes) %#v", s)
log.Print(err1); return "", err1}}
func import_ObjectCannedACL(s string) (types.ObjectCannedACL, error) {
switch s {
case "private": return types.ObjectCannedACLPrivate, nil
case "public-read": return types.ObjectCannedACLPublicRead, nil
case "public-read-write": return types.ObjectCannedACLPublicReadWrite, nil
case "authenticated-read": return types.ObjectCannedACLAuthenticatedRead, nil
case "aws-exec-read": return types.ObjectCannedACLAwsExecRead, nil
case "bucket-owner-read": return types.ObjectCannedACLBucketOwnerRead, nil
case "bucket-owner-full-control": return types.ObjectCannedACLBucketOwnerFullControl, nil
default: var err1 = fmt.Errorf("interning an enum (types.ObjectCannedACL) %#v", s)
log.Print(err1); return "", err1}}
func import_ObjectLockLegalHoldStatus(s string) (types.ObjectLockLegalHoldStatus, error) {
switch s {
case "ON": return types.ObjectLockLegalHoldStatusOn, nil
case "OFF": return types.ObjectLockLegalHoldStatusOff, nil
default: var err1 = fmt.Errorf("interning an enum (types.ObjectLockLegalHoldStatus) %#v", s)
log.Print(err1); return "", err1}}
func import_ObjectLockMode(s string) (types.ObjectLockMode, error) {
switch s {
case "GOVERNANCE": return types.ObjectLockModeGovernance, nil
case "COMPLIANCE": return types.ObjectLockModeCompliance, nil
default: var err1 = fmt.Errorf("interning an enum (types.ObjectLockMode) %#v", s)
log.Print(err1); return "", err1}}
func import_ObjectOwnership(s string) (types.ObjectOwnership, error) {
switch s {
case "BucketOwnerPreferred": return types.ObjectOwnershipBucketOwnerPreferred, nil
case "ObjectWriter": return types.ObjectOwnershipObjectWriter, nil
case "BucketOwnerEnforced": return types.ObjectOwnershipBucketOwnerEnforced, nil
default: var err1 = fmt.Errorf("interning an enum (types.ObjectOwnership) %#v", s)
log.Print(err1); return "", err1}}
func import_OptionalObjectAttributes(s string) (types.OptionalObjectAttributes, error) {
switch s {
case "RestoreStatus": return types.OptionalObjectAttributesRestoreStatus, nil
default: var err1 = fmt.Errorf("interning an enum (types.OptionalObjectAttributes) %#v", s)
log.Print(err1); return "", err1}}
func import_RequestPayer(s string) (types.RequestPayer, error) {
switch s {
case "requester": return types.RequestPayerRequester, nil
default: var err1 = fmt.Errorf("interning an enum (types.RequestPayer) %#v", s)
log.Print(err1); return "", err1}}
func import_ServerSideEncryption(s string) (types.ServerSideEncryption, error) {
switch s {
case "AES256": return types.ServerSideEncryptionAes256, nil
case "aws:fsx": return types.ServerSideEncryptionAwsFsx, nil
case "aws:kms": return types.ServerSideEncryptionAwsKms, nil
case "aws:kms:dsse": return types.ServerSideEncryptionAwsKmsDsse, nil
default: var err1 = fmt.Errorf("interning an enum (types.ServerSideEncryption) %#v", s)
log.Print(err1); return "", err1}}
func import_StorageClass(s string) (types.StorageClass, error) {
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
default: var err1 = fmt.Errorf("interning an enum (types.StorageClass) %#v", s)
log.Print(err1); return "", err1}}
func import_TaggingDirective(s string) (types.TaggingDirective, error) {
switch s {
case "COPY": return types.TaggingDirectiveCopy, nil
case "REPLACE": return types.TaggingDirectiveReplace, nil
default: var err1 = fmt.Errorf("interning an enum (types.TaggingDirective) %#v", s)
log.Print(err1); return "", err1}}

