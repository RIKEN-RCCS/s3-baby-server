// marshalers.go (2025-11-02)
// API-STUB.  Marshalers of response structures.  Response
// structures need custom marshalers, because they have
// some slots that need to be renamed and also have an
// extra slot that should be suppressed.
package server
import (
"encoding/xml"
"github.com/aws/aws-sdk-go-v2/service/s3"
)
func thing_pointer[T any](v T) *T {return &v}
func start_element(k string) xml.StartElement {
return xml.StartElement{Name: xml.Name{Local: k}}
}
type s_AbortMultipartUploadResponse s3.AbortMultipartUploadOutput
func (s s_AbortMultipartUploadResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_CompleteMultipartUploadResponse s3.CompleteMultipartUploadOutput
func (s s_CompleteMultipartUploadResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Location, start_element("Location"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Bucket, start_element("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Key, start_element("Key"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ETag, start_element("ETag"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumCRC32, start_element("ChecksumCRC32"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumCRC32C, start_element("ChecksumCRC32C"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumCRC64NVME, start_element("ChecksumCRC64NVME"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumSHA1, start_element("ChecksumSHA1"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumSHA256, start_element("ChecksumSHA256"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumType, start_element("ChecksumType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_CopyObjectResponse s3.CopyObjectOutput
func (s s_CopyObjectResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyObjectResult, start_element("CopyObjectResult"))
if err2 != nil {return err2}}
return nil}
type s_CreateBucketResponse s3.CreateBucketOutput
func (s s_CreateBucketResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_CreateMultipartUploadResponse s3.CreateMultipartUploadOutput
func (s s_CreateMultipartUploadResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Bucket, start_element("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Key, start_element("Key"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.UploadId, start_element("UploadId"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_DeleteObjectResponse s3.DeleteObjectOutput
func (s s_DeleteObjectResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_DeleteObjectsResponse s3.DeleteObjectsOutput
func (s s_DeleteObjectsResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Deleted, start_element("Deleted"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Errors, start_element("Error"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_DeleteObjectTaggingResponse s3.DeleteObjectTaggingOutput
func (s s_DeleteObjectTaggingResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_GetObjectResponse s3.GetObjectOutput
func (s s_GetObjectResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
{var err2 = e.EncodeElement(s.Body, start_element("Body"))
if err2 != nil {return err2}}
return nil}
type s_GetObjectAttributesResponse s3.GetObjectAttributesOutput
func (s s_GetObjectAttributesResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.ETag, start_element("ETag"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Checksum, start_element("Checksum"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ObjectParts, start_element("ObjectParts"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.StorageClass, start_element("StorageClass"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ObjectSize, start_element("ObjectSize"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_GetObjectTaggingResponse s3.GetObjectTaggingOutput
func (s s_GetObjectTaggingResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.TagSet, start_element("TagSet"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_HeadBucketResponse s3.HeadBucketOutput
func (s s_HeadBucketResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_HeadObjectResponse s3.HeadObjectOutput
func (s s_HeadObjectResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_ListBucketsResponse s3.ListBucketsOutput
func (s s_ListBucketsResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Buckets, start_element("Buckets"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Owner, start_element("Owner"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ContinuationToken, start_element("ContinuationToken"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, start_element("Prefix"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_ListMultipartUploadsResponse s3.ListMultipartUploadsOutput
func (s s_ListMultipartUploadsResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Bucket, start_element("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.KeyMarker, start_element("KeyMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.UploadIdMarker, start_element("UploadIdMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextKeyMarker, start_element("NextKeyMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, start_element("Prefix"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Delimiter, start_element("Delimiter"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextUploadIdMarker, start_element("NextUploadIdMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxUploads, start_element("MaxUploads"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.IsTruncated, start_element("IsTruncated"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Uploads, start_element("Upload"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.CommonPrefixes, start_element("CommonPrefixes"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.EncodingType, start_element("EncodingType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_ListObjectsResponse s3.ListObjectsOutput
func (s s_ListObjectsResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.IsTruncated, start_element("IsTruncated"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Marker, start_element("Marker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextMarker, start_element("NextMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Contents, start_element("Contents"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Name, start_element("Name"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, start_element("Prefix"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Delimiter, start_element("Delimiter"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxKeys, start_element("MaxKeys"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.CommonPrefixes, start_element("CommonPrefixes"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.EncodingType, start_element("EncodingType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_ListObjectsV2Response s3.ListObjectsV2Output
func (s s_ListObjectsV2Response) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.IsTruncated, start_element("IsTruncated"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Contents, start_element("Contents"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Name, start_element("Name"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, start_element("Prefix"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Delimiter, start_element("Delimiter"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxKeys, start_element("MaxKeys"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.CommonPrefixes, start_element("CommonPrefixes"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.EncodingType, start_element("EncodingType"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.KeyCount, start_element("KeyCount"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ContinuationToken, start_element("ContinuationToken"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextContinuationToken, start_element("NextContinuationToken"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.StartAfter, start_element("StartAfter"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_ListPartsResponse s3.ListPartsOutput
func (s s_ListPartsResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
var err1 = e.EncodeToken(start)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Bucket, start_element("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Key, start_element("Key"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.UploadId, start_element("UploadId"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.PartNumberMarker, start_element("PartNumberMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextPartNumberMarker, start_element("NextPartNumberMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxParts, start_element("MaxParts"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.IsTruncated, start_element("IsTruncated"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Parts, start_element("Part"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Initiator, start_element("Initiator"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Owner, start_element("Owner"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.StorageClass, start_element("StorageClass"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumAlgorithm, start_element("ChecksumAlgorithm"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumType, start_element("ChecksumType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(start.End())
if err9 != nil {return err9}
return nil}
type s_PutObjectResponse s3.PutObjectOutput
func (s s_PutObjectResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_PutObjectTaggingResponse s3.PutObjectTaggingOutput
func (s s_PutObjectTaggingResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_UploadPartResponse s3.UploadPartOutput
func (s s_UploadPartResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
return nil}
type s_UploadPartCopyResponse s3.UploadPartCopyOutput
func (s s_UploadPartCopyResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyPartResult, start_element("CopyPartResult"))
if err2 != nil {return err2}}
return nil}
