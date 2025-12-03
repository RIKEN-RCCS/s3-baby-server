// marshaler.go (2025-12-03)
// API-STUB.  Marshalers of response structures.  Response
// structures need custom marshalers, because they have
// some slots that need to be renamed and also have an
// extra slot that should be suppressed.

package server
import (
"encoding/xml"
"github.com/aws/aws-sdk-go-v2/service/s3"
)
func h_thing_pointer[T any](v T) *T {return &v}
func h_make_tag(k string) xml.StartElement {
return xml.StartElement{Name: xml.Name{Local: k}}}
type h_CompleteMultipartUploadResponse s3.CompleteMultipartUploadOutput
func (s h_CompleteMultipartUploadResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("CompleteMultipartUploadResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.Location != nil {
var err2 = e.EncodeElement(s.Location, h_make_tag("Location"))
if err2 != nil {return err2}}
if s.Bucket != nil {
var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
if s.Key != nil {
var err2 = e.EncodeElement(s.Key, h_make_tag("Key"))
if err2 != nil {return err2}}
if s.ETag != nil {
var err2 = e.EncodeElement(s.ETag, h_make_tag("ETag"))
if err2 != nil {return err2}}
if s.ChecksumCRC32 != nil {
var err2 = e.EncodeElement(s.ChecksumCRC32, h_make_tag("ChecksumCRC32"))
if err2 != nil {return err2}}
if s.ChecksumCRC32C != nil {
var err2 = e.EncodeElement(s.ChecksumCRC32C, h_make_tag("ChecksumCRC32C"))
if err2 != nil {return err2}}
if s.ChecksumCRC64NVME != nil {
var err2 = e.EncodeElement(s.ChecksumCRC64NVME, h_make_tag("ChecksumCRC64NVME"))
if err2 != nil {return err2}}
if s.ChecksumSHA1 != nil {
var err2 = e.EncodeElement(s.ChecksumSHA1, h_make_tag("ChecksumSHA1"))
if err2 != nil {return err2}}
if s.ChecksumSHA256 != nil {
var err2 = e.EncodeElement(s.ChecksumSHA256, h_make_tag("ChecksumSHA256"))
if err2 != nil {return err2}}
if s.ChecksumType != "" {
var err2 = e.EncodeElement(s.ChecksumType, h_make_tag("ChecksumType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_CopyObjectResponse s3.CopyObjectOutput
func (s h_CopyObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyObjectResult, h_make_tag("CopyObjectResult"))
if err2 != nil {return err2}}
return nil}
type h_CreateMultipartUploadResponse s3.CreateMultipartUploadOutput
func (s h_CreateMultipartUploadResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("InitiateMultipartUploadResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.Bucket != nil {
var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
if s.Key != nil {
var err2 = e.EncodeElement(s.Key, h_make_tag("Key"))
if err2 != nil {return err2}}
if s.UploadId != nil {
var err2 = e.EncodeElement(s.UploadId, h_make_tag("UploadId"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_DeleteObjectsResponse s3.DeleteObjectsOutput
func (s h_DeleteObjectsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("DeleteResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.Deleted != nil {
var err2 = e.EncodeElement(s.Deleted, h_make_tag("Deleted"))
if err2 != nil {return err2}}
if s.Errors != nil {
var err2 = e.EncodeElement(s.Errors, h_make_tag("Error"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_GetObjectResponse s3.GetObjectOutput
func (s h_GetObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.Body, h_make_tag("Body"))
if err2 != nil {return err2}}
return nil}
type h_GetObjectAttributesResponse s3.GetObjectAttributesOutput
func (s h_GetObjectAttributesResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("GetObjectAttributesResponse")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.ETag != nil {
var err2 = e.EncodeElement(s.ETag, h_make_tag("ETag"))
if err2 != nil {return err2}}
if s.Checksum != nil {
var err2 = e.EncodeElement(s.Checksum, h_make_tag("Checksum"))
if err2 != nil {return err2}}
if s.ObjectParts != nil {
var err2 = e.EncodeElement(s.ObjectParts, h_make_tag("ObjectParts"))
if err2 != nil {return err2}}
if s.StorageClass != "" {
var err2 = e.EncodeElement(s.StorageClass, h_make_tag("StorageClass"))
if err2 != nil {return err2}}
if s.ObjectSize != nil {
var err2 = e.EncodeElement(s.ObjectSize, h_make_tag("ObjectSize"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_GetObjectTaggingResponse s3.GetObjectTaggingOutput
func (s h_GetObjectTaggingResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("Tagging")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.TagSet != nil {
var err2 = e.EncodeElement(s.TagSet, h_make_tag("TagSet"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListBucketsResponse s3.ListBucketsOutput
func (s h_ListBucketsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListAllMyBucketsResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.Buckets != nil {
var err2 = e.EncodeElement(s.Buckets, h_make_tag("Buckets"))
if err2 != nil {return err2}}
if s.Owner != nil {
var err2 = e.EncodeElement(s.Owner, h_make_tag("Owner"))
if err2 != nil {return err2}}
if s.ContinuationToken != nil {
var err2 = e.EncodeElement(s.ContinuationToken, h_make_tag("ContinuationToken"))
if err2 != nil {return err2}}
if s.Prefix != nil {
var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListMultipartUploadsResponse s3.ListMultipartUploadsOutput
func (s h_ListMultipartUploadsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListMultipartUploadsResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.Bucket != nil {
var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
if s.KeyMarker != nil {
var err2 = e.EncodeElement(s.KeyMarker, h_make_tag("KeyMarker"))
if err2 != nil {return err2}}
if s.UploadIdMarker != nil {
var err2 = e.EncodeElement(s.UploadIdMarker, h_make_tag("UploadIdMarker"))
if err2 != nil {return err2}}
if s.NextKeyMarker != nil {
var err2 = e.EncodeElement(s.NextKeyMarker, h_make_tag("NextKeyMarker"))
if err2 != nil {return err2}}
if s.Prefix != nil {
var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
if s.Delimiter != nil {
var err2 = e.EncodeElement(s.Delimiter, h_make_tag("Delimiter"))
if err2 != nil {return err2}}
if s.NextUploadIdMarker != nil {
var err2 = e.EncodeElement(s.NextUploadIdMarker, h_make_tag("NextUploadIdMarker"))
if err2 != nil {return err2}}
if s.MaxUploads != nil {
var err2 = e.EncodeElement(s.MaxUploads, h_make_tag("MaxUploads"))
if err2 != nil {return err2}}
if s.IsTruncated != nil {
var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
if s.Uploads != nil {
var err2 = e.EncodeElement(s.Uploads, h_make_tag("Upload"))
if err2 != nil {return err2}}
if s.CommonPrefixes != nil {
var err2 = e.EncodeElement(s.CommonPrefixes, h_make_tag("CommonPrefixes"))
if err2 != nil {return err2}}
if s.EncodingType != "" {
var err2 = e.EncodeElement(s.EncodingType, h_make_tag("EncodingType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListObjectsResponse s3.ListObjectsOutput
func (s h_ListObjectsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListBucketResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.IsTruncated != nil {
var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
if s.Marker != nil {
var err2 = e.EncodeElement(s.Marker, h_make_tag("Marker"))
if err2 != nil {return err2}}
if s.NextMarker != nil {
var err2 = e.EncodeElement(s.NextMarker, h_make_tag("NextMarker"))
if err2 != nil {return err2}}
if s.Contents != nil {
var err2 = e.EncodeElement(s.Contents, h_make_tag("Contents"))
if err2 != nil {return err2}}
if s.Name != nil {
var err2 = e.EncodeElement(s.Name, h_make_tag("Name"))
if err2 != nil {return err2}}
if s.Prefix != nil {
var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
if s.Delimiter != nil {
var err2 = e.EncodeElement(s.Delimiter, h_make_tag("Delimiter"))
if err2 != nil {return err2}}
if s.MaxKeys != nil {
var err2 = e.EncodeElement(s.MaxKeys, h_make_tag("MaxKeys"))
if err2 != nil {return err2}}
if s.CommonPrefixes != nil {
var err2 = e.EncodeElement(s.CommonPrefixes, h_make_tag("CommonPrefixes"))
if err2 != nil {return err2}}
if s.EncodingType != "" {
var err2 = e.EncodeElement(s.EncodingType, h_make_tag("EncodingType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListObjectsV2Response s3.ListObjectsV2Output
func (s h_ListObjectsV2Response) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListBucketResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.IsTruncated != nil {
var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
if s.Contents != nil {
var err2 = e.EncodeElement(s.Contents, h_make_tag("Contents"))
if err2 != nil {return err2}}
if s.Name != nil {
var err2 = e.EncodeElement(s.Name, h_make_tag("Name"))
if err2 != nil {return err2}}
if s.Prefix != nil {
var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
if s.Delimiter != nil {
var err2 = e.EncodeElement(s.Delimiter, h_make_tag("Delimiter"))
if err2 != nil {return err2}}
if s.MaxKeys != nil {
var err2 = e.EncodeElement(s.MaxKeys, h_make_tag("MaxKeys"))
if err2 != nil {return err2}}
if s.CommonPrefixes != nil {
var err2 = e.EncodeElement(s.CommonPrefixes, h_make_tag("CommonPrefixes"))
if err2 != nil {return err2}}
if s.EncodingType != "" {
var err2 = e.EncodeElement(s.EncodingType, h_make_tag("EncodingType"))
if err2 != nil {return err2}}
if s.KeyCount != nil {
var err2 = e.EncodeElement(s.KeyCount, h_make_tag("KeyCount"))
if err2 != nil {return err2}}
if s.ContinuationToken != nil {
var err2 = e.EncodeElement(s.ContinuationToken, h_make_tag("ContinuationToken"))
if err2 != nil {return err2}}
if s.NextContinuationToken != nil {
var err2 = e.EncodeElement(s.NextContinuationToken, h_make_tag("NextContinuationToken"))
if err2 != nil {return err2}}
if s.StartAfter != nil {
var err2 = e.EncodeElement(s.StartAfter, h_make_tag("StartAfter"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListPartsResponse s3.ListPartsOutput
func (s h_ListPartsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListPartsResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
if s.Bucket != nil {
var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
if s.Key != nil {
var err2 = e.EncodeElement(s.Key, h_make_tag("Key"))
if err2 != nil {return err2}}
if s.UploadId != nil {
var err2 = e.EncodeElement(s.UploadId, h_make_tag("UploadId"))
if err2 != nil {return err2}}
if s.PartNumberMarker != nil {
var err2 = e.EncodeElement(s.PartNumberMarker, h_make_tag("PartNumberMarker"))
if err2 != nil {return err2}}
if s.NextPartNumberMarker != nil {
var err2 = e.EncodeElement(s.NextPartNumberMarker, h_make_tag("NextPartNumberMarker"))
if err2 != nil {return err2}}
if s.MaxParts != nil {
var err2 = e.EncodeElement(s.MaxParts, h_make_tag("MaxParts"))
if err2 != nil {return err2}}
if s.IsTruncated != nil {
var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
if s.Parts != nil {
var err2 = e.EncodeElement(s.Parts, h_make_tag("Part"))
if err2 != nil {return err2}}
if s.Initiator != nil {
var err2 = e.EncodeElement(s.Initiator, h_make_tag("Initiator"))
if err2 != nil {return err2}}
if s.Owner != nil {
var err2 = e.EncodeElement(s.Owner, h_make_tag("Owner"))
if err2 != nil {return err2}}
if s.StorageClass != "" {
var err2 = e.EncodeElement(s.StorageClass, h_make_tag("StorageClass"))
if err2 != nil {return err2}}
if s.ChecksumAlgorithm != "" {
var err2 = e.EncodeElement(s.ChecksumAlgorithm, h_make_tag("ChecksumAlgorithm"))
if err2 != nil {return err2}}
if s.ChecksumType != "" {
var err2 = e.EncodeElement(s.ChecksumType, h_make_tag("ChecksumType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_UploadPartCopyResponse s3.UploadPartCopyOutput
func (s h_UploadPartCopyResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyPartResult, h_make_tag("CopyPartResult"))
if err2 != nil {return err2}}
return nil}
