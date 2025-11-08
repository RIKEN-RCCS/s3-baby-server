// marshalers.go (2025-11-08)
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
type h_AbortMultipartUploadResponse s3.AbortMultipartUploadOutput
func (s h_AbortMultipartUploadResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_CompleteMultipartUploadResponse s3.CompleteMultipartUploadOutput
func (s h_CompleteMultipartUploadResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("CompleteMultipartUploadResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Location, h_make_tag("Location"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Key, h_make_tag("Key"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ETag, h_make_tag("ETag"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumCRC32, h_make_tag("ChecksumCRC32"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumCRC32C, h_make_tag("ChecksumCRC32C"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumCRC64NVME, h_make_tag("ChecksumCRC64NVME"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumSHA1, h_make_tag("ChecksumSHA1"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumSHA256, h_make_tag("ChecksumSHA256"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumType, h_make_tag("ChecksumType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_CopyObjectResponse s3.CopyObjectOutput
func (s h_CopyObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyObjectResult, h_make_tag("CopyObjectResult"))
if err2 != nil {return err2}}
return nil}
type h_CreateBucketResponse s3.CreateBucketOutput
func (s h_CreateBucketResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_CreateMultipartUploadResponse s3.CreateMultipartUploadOutput
func (s h_CreateMultipartUploadResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("InitiateMultipartUploadResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Key, h_make_tag("Key"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.UploadId, h_make_tag("UploadId"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_DeleteObjectResponse s3.DeleteObjectOutput
func (s h_DeleteObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_DeleteObjectsResponse s3.DeleteObjectsOutput
func (s h_DeleteObjectsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("DeleteResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var tag2 = h_make_tag("Deleted")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.Deleted)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var tag2 = h_make_tag("Error")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.Errors)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_DeleteObjectTaggingResponse s3.DeleteObjectTaggingOutput
func (s h_DeleteObjectTaggingResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
{var err2 = e.EncodeElement(s.ETag, h_make_tag("ETag"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Checksum, h_make_tag("Checksum"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ObjectParts, h_make_tag("ObjectParts"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.StorageClass, h_make_tag("StorageClass"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ObjectSize, h_make_tag("ObjectSize"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_GetObjectTaggingResponse s3.GetObjectTaggingOutput
func (s h_GetObjectTaggingResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("Tagging")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var tag2 = h_make_tag("TagSet")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.TagSet)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_HeadBucketResponse s3.HeadBucketOutput
func (s h_HeadBucketResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_HeadObjectResponse s3.HeadObjectOutput
func (s h_HeadObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_ListBucketsResponse s3.ListBucketsOutput
func (s h_ListBucketsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListAllMyBucketsResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var tag2 = h_make_tag("Buckets")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.Buckets)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var err2 = e.EncodeElement(s.Owner, h_make_tag("Owner"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ContinuationToken, h_make_tag("ContinuationToken"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListMultipartUploadsResponse s3.ListMultipartUploadsOutput
func (s h_ListMultipartUploadsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListMultipartUploadsResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.KeyMarker, h_make_tag("KeyMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.UploadIdMarker, h_make_tag("UploadIdMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextKeyMarker, h_make_tag("NextKeyMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Delimiter, h_make_tag("Delimiter"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextUploadIdMarker, h_make_tag("NextUploadIdMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxUploads, h_make_tag("MaxUploads"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
{var tag2 = h_make_tag("Upload")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.Uploads)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var tag2 = h_make_tag("CommonPrefixes")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.CommonPrefixes)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var err2 = e.EncodeElement(s.EncodingType, h_make_tag("EncodingType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListObjectsResponse s3.ListObjectsOutput
func (s h_ListObjectsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListBucketResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Marker, h_make_tag("Marker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextMarker, h_make_tag("NextMarker"))
if err2 != nil {return err2}}
{var tag2 = h_make_tag("Contents")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.Contents)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var err2 = e.EncodeElement(s.Name, h_make_tag("Name"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Delimiter, h_make_tag("Delimiter"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxKeys, h_make_tag("MaxKeys"))
if err2 != nil {return err2}}
{var tag2 = h_make_tag("CommonPrefixes")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.CommonPrefixes)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var err2 = e.EncodeElement(s.EncodingType, h_make_tag("EncodingType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListObjectsV2Response s3.ListObjectsV2Output
func (s h_ListObjectsV2Response) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListBucketResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
{var tag2 = h_make_tag("Contents")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.Contents)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var err2 = e.EncodeElement(s.Name, h_make_tag("Name"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Prefix, h_make_tag("Prefix"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Delimiter, h_make_tag("Delimiter"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxKeys, h_make_tag("MaxKeys"))
if err2 != nil {return err2}}
{var tag2 = h_make_tag("CommonPrefixes")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.CommonPrefixes)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var err2 = e.EncodeElement(s.EncodingType, h_make_tag("EncodingType"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.KeyCount, h_make_tag("KeyCount"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ContinuationToken, h_make_tag("ContinuationToken"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextContinuationToken, h_make_tag("NextContinuationToken"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.StartAfter, h_make_tag("StartAfter"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_ListPartsResponse s3.ListPartsOutput
func (s h_ListPartsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListPartsResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
{var err2 = e.EncodeElement(s.Bucket, h_make_tag("Bucket"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Key, h_make_tag("Key"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.UploadId, h_make_tag("UploadId"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.PartNumberMarker, h_make_tag("PartNumberMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.NextPartNumberMarker, h_make_tag("NextPartNumberMarker"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.MaxParts, h_make_tag("MaxParts"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.IsTruncated, h_make_tag("IsTruncated"))
if err2 != nil {return err2}}
{var tag2 = h_make_tag("Part")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
var err3 = e.Encode(s.Parts)
if err3 != nil {return err3}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}}
{var err2 = e.EncodeElement(s.Initiator, h_make_tag("Initiator"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.Owner, h_make_tag("Owner"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.StorageClass, h_make_tag("StorageClass"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumAlgorithm, h_make_tag("ChecksumAlgorithm"))
if err2 != nil {return err2}}
{var err2 = e.EncodeElement(s.ChecksumType, h_make_tag("ChecksumType"))
if err2 != nil {return err2}}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type h_PutObjectResponse s3.PutObjectOutput
func (s h_PutObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_PutObjectTaggingResponse s3.PutObjectTaggingOutput
func (s h_PutObjectTaggingResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_UploadPartResponse s3.UploadPartOutput
func (s h_UploadPartResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
return nil}
type h_UploadPartCopyResponse s3.UploadPartCopyOutput
func (s h_UploadPartCopyResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyPartResult, h_make_tag("CopyPartResult"))
if err2 != nil {return err2}}
return nil}
