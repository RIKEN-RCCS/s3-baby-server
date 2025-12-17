// marshaler.go (2025-12-18)
// API-STUB.  Marshalers of response structures.  Response
// structures need custom marshalers, because they have
// some slots that need to be renamed and also have an
// extra slot that should be suppressed.

package server
import (
"bytes"
"crypto/md5"
"encoding/base64"
"encoding/xml"
"fmt"
"github.com/aws/aws-sdk-go-v2/service/s3"
"github.com/aws/aws-sdk-go-v2/service/s3/types"
"io"
"net/http"
)
func h_thing_pointer[T any](v T) *T {return &v}
func h_make_tag(k string) xml.StartElement {
return xml.StartElement{Name: xml.Name{Local: k}}}
// H_DECODE_BODY decodes the XML body with checking its md5 hash,
// when a header exists.
func h_decode_body(x any, body io.Reader, h http.Header) error {
var hash = md5.New()
var body2 = &io.LimitedReader{R: body, N: h_xml_body_limit}
var r io.Reader = io.TeeReader(body2, hash)
var d = xml.NewDecoder(r)
var err1 = d.Decode(x)
if err1 != nil {return err1}
var md5c = hash.Sum(nil)
var s = h.Get("Content-MD5")
if s != "" {
var md5h, err2 = base64.StdEncoding.DecodeString(s)
if err2 != nil {return err2}
if bytes.Compare(md5c, md5h) != 0 {
return fmt.Errorf("MD5 mismatch")}}
return nil}
type O_CompleteMultipartUploadResponse s3.CompleteMultipartUploadOutput
func (s O_CompleteMultipartUploadResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_CopyObjectResponse s3.CopyObjectOutput
func (s O_CopyObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyObjectResult, h_make_tag("CopyObjectResult"))
if err2 != nil {return err2}}
return nil}
type O_CreateMultipartUploadResponse s3.CreateMultipartUploadOutput
func (s O_CreateMultipartUploadResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_DeleteObjectsResponse s3.DeleteObjectsOutput
func (s O_DeleteObjectsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_GetObjectResponse s3.GetObjectOutput
func (s O_GetObjectResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.Body, h_make_tag("Body"))
if err2 != nil {return err2}}
return nil}
type O_GetObjectAttributesResponse s3.GetObjectAttributesOutput
func (s O_GetObjectAttributesResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_GetObjectTaggingResponse s3.GetObjectTaggingOutput
func (s O_GetObjectTaggingResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("Tagging")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
// XML TAG-AFFIX.
var tag2 = h_make_tag("TagSet")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
if s.TagSet != nil {
var err3 = e.EncodeElement(s.TagSet, h_make_tag("Tag"))
if err3 != nil {return err3}}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}
var err9 = e.EncodeToken(tag1.End())
if err9 != nil {return err9}
return nil}
type O_ListBucketsResponse s3.ListBucketsOutput
func (s O_ListBucketsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
var tag1 = h_make_tag("ListAllMyBucketsResult")
var err1 = e.EncodeToken(tag1)
if err1 != nil {return err1}
// XML TAG-AFFIX.
var tag2 = h_make_tag("Buckets")
var err2 = e.EncodeToken(tag2)
if err2 != nil {return err2}
if s.Buckets != nil {
var err3 = e.EncodeElement(s.Buckets, h_make_tag("Bucket"))
if err3 != nil {return err3}}
var err4 = e.EncodeToken(tag2.End())
if err4 != nil {return err4}
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
type O_ListMultipartUploadsResponse s3.ListMultipartUploadsOutput
func (s O_ListMultipartUploadsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_ListObjectsResponse s3.ListObjectsOutput
func (s O_ListObjectsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_ListObjectsV2Response s3.ListObjectsV2Output
func (s O_ListObjectsV2Response) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_ListPartsResponse s3.ListPartsOutput
func (s O_ListPartsResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
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
type O_UploadPartCopyResponse s3.UploadPartCopyOutput
func (s O_UploadPartCopyResponse) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
{var err2 = e.EncodeElement(s.CopyPartResult, h_make_tag("CopyPartResult"))
if err2 != nil {return err2}}
return nil}
type O_CompletedMultipartUpload struct {
XMLName xml.Name `xml:"CompleteMultipartUpload"`
Part []types.CompletedPart
}
func import_CompletedMultipartUpload(o *O_CompletedMultipartUpload) *types.CompletedMultipartUpload {
var i = types.CompletedMultipartUpload{
Parts: o.Part,
}
return &i}
type O_CreateBucketConfiguration struct {
XMLName xml.Name `xml:"CreateBucketConfiguration"`
LocationConstraint types.BucketLocationConstraint
Location *types.LocationInfo
Bucket *types.BucketInfo
Tags struct {Tag []types.Tag}
}
func import_CreateBucketConfiguration(o *O_CreateBucketConfiguration) *types.CreateBucketConfiguration {
var i = types.CreateBucketConfiguration{
LocationConstraint: o.LocationConstraint,
Location: o.Location,
Bucket: o.Bucket,
Tags: o.Tags.Tag,
}
return &i}
type O_Delete struct {
XMLName xml.Name `xml:"Delete"`
Object []types.ObjectIdentifier
Quiet *bool
}
func import_Delete(o *O_Delete) *types.Delete {
var i = types.Delete{
Objects: o.Object,
Quiet: o.Quiet,
}
return &i}
type O_Tagging struct {
XMLName xml.Name `xml:"Tagging"`
TagSet struct {Tag []types.Tag}
}
func import_Tagging(o *O_Tagging) *types.Tagging {
var i = types.Tagging{
TagSet: o.TagSet.Tag,
}
return &i}
