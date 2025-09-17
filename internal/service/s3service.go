// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"s3-baby-server/internal/model"
	"s3-baby-server/pkg/utils"
	"strings"
)

type S3Service struct {
	FileSystem *FileSystem
	MultiPart  *MultiPart
	Tag        *Tag
}

func (s3 *S3Service) AbortMultipartUpload(option S3Options) *S3Error {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return InvalidBucketName()
	}
	if !s3.FileSystem.isFileExists(option.GetBucket()) {
		return NoSuchBucket()
	}
	id := option.GetOption("uploadId")
	if !s3.MultiPart.uploadIDExists(id) {
		return NoSuchUpload()
	}
	if !s3.FileSystem.forceDeleteDir(id) {
		return InternalError()
	}
	return nil
}

func (s3 *S3Service) CompleteMultipartUpload(option S3Options) (*model.CompleteMultipartUploadResult, *S3Error) {
	var s3err *S3Error
	s := model.CompleteMultipartUploadState{}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if s.Bucket = option.GetBucket(); !s3.FileSystem.isFileExists(s.Bucket) {
		return nil, NoSuchBucket()
	}
	s.Key = option.GetKey()
	if !s3.FileSystem.checkKeyName(s.Key) {
		return nil, KeyTooLongError()
	}
	id := option.GetOption("uploadId")
	if !s3.MultiPart.uploadIDExists(id) {
		return nil, NoSuchUpload()
	}
	var reqBody model.CompleteMultipartUploadRequest
	if err := xml.Unmarshal(option.GetBody(), &reqBody); err != nil {
		return nil, InvalidRequest()
	}
	if !s3.checkPartSize(id, reqBody) {
		return nil, EntityTooSmallError()
	}
	s.DstPath = option.GetPath()
	if err := s3.MultiPart.completeMpUpload(s.Key, id, s.DstPath, reqBody); err != nil {
		return nil, err
	}
	if s3err = s3.checkObjectSize(option, s.DstPath); s3err != nil {
		return nil, s3err
	}
	if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.getChecksumMode(option, ""); s3err != nil {
		return nil, s3err
	}
	if s.ETag, s3err = s3.getETag(s.DstPath); s3err != nil {
		return nil, s3err
	}
	metaAPath := s3.FileSystem.getFullPath(s3.FileSystem.getPartNumberPath(id, s.Key)) // タグが設定されている場合は指定の場所に移動
	if s3.FileSystem.isFileExists(s3.FileSystem.getMetaFileName(metaAPath)) {
		s3.FileSystem.moveFile(s3.FileSystem.getMetaFileName(metaAPath), s3.FileSystem.getMetaFileName(option.GetPath()))
	}
	result := s.MakeCompleteMultipartUploadResult()
	s3.FileSystem.forceDeleteDir(id)
	return result, nil
}

func (s3 *S3Service) CopyObject(option S3Options) (*model.CopyObjectResult, *S3Error) {
	s := model.CopyObjectState{}
	s.SrcPath = option.GetOption("x-amz-copy-source")
	if s.SrcPath == "" {
		return nil, InvalidArgument()
	}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if err := s3.isBucketAndKeyExists(option.GetBucket(), s.SrcPath); err != nil {
		return nil, err
	}
	if err := s3.validateOptions(option); err != nil {
		return nil, err
	}
	if !s3.needsCopy(option, s.SrcPath) {
		return nil, PreconditionFailed()
	}
	s.DstPath = option.GetPath()
	if err := s3.Tag.taggingDirective(option, s.SrcPath, s.DstPath); err != nil {
		return nil, err
	}
	if !s3.FileSystem.copyFile(s.SrcPath, s.DstPath, 0, 0) {
		return nil, InternalError()
	}
	var s3err *S3Error
	if s.Info, s.ETag, s3err = s3.getFileInfoAndETag(s.DstPath); s3err != nil {
		return nil, s3err
	}
	a := option.GetOption("x-amz-checksum-algorithm")
	if !s3.FileSystem.validateChecksumAlgorithm(a) {
		return nil, InvalidArgument()
	}
	if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.compareChecksum(option, s.SrcPath, a, "", true); s3err != nil {
		return nil, s3err
	}
	result := s.MakeCopyObjectResult()
	return result, nil
}

func (s3 *S3Service) CreateBucket(option S3Options) (string, *S3Error) {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return "", InvalidBucketName()
	}
	if err := s3.validateOption(option); err != nil {
		return "", err
	}
	if s3.FileSystem.isFileExists(option.GetBucket()) {
		return "", BucketAlreadyOwnedByYou()
	}
	if err := s3.FileSystem.createDir(option.GetBucket()); err != nil {
		return "", InternalError()
	}
	return option.GetBucket(), nil
}

func (s3 *S3Service) CreateMultipartUpload(option S3Options) (*model.CreateMultipartUploadResult, *S3Error) {
	s := model.CreateMultipartUploadState{}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if s.Bucket = option.GetBucket(); !s3.FileSystem.isFileExists(s.Bucket) {
		return nil, NoSuchBucket()
	}
	s.Key = option.GetKey()
	if !s3.FileSystem.checkKeyName(s.Key) {
		return nil, KeyTooLongError()
	}
	var s3err *S3Error
	if s3err = s3.validateOptions(option); s3err != nil {
		return nil, s3err
	}
	if !s3.FileSystem.validateChecksumAlgorithm(option.GetOption("x-amz-checksum-algorithm")) {
		return nil, InvalidArgument()
	}
	if s.UploadID = s3.MultiPart.makeUploadID(); s.UploadID == -1 {
		return nil, InternalError()
	}
	result := s.MakeCreateMultipartUploadResult()
	if !s3.MultiPart.createMpUploadMeta(*result) {
		return nil, InternalError()
	}
	if s3err = s3.Tag.putTagging(option, s3.FileSystem.getPartNumberPath(utils.ToString(s.UploadID), s.Key)); s3err != nil {
		return nil, s3err
	}
	return result, nil
}

func (s3 *S3Service) DeleteBucket(option S3Options) *S3Error {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return InvalidBucketName()
	}
	if !s3.FileSystem.isFileExists(option.GetBucket()) {
		return NoSuchBucket()
	}
	if !s3.FileSystem.deleteFile(option.GetBucket()) {
		return BucketNotEmpty()
	}
	return nil
}

func (s3 *S3Service) DeleteObject(option S3Options) *S3Error {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return InvalidBucketName()
	}
	if b := option.GetBucket(); !s3.FileSystem.isFileExists(b) {
		return NoSuchBucket()
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return KeyTooLongError()
	}
	if isDir := strings.HasSuffix(option.GetKey(), "/"); isDir {
		if err := os.RemoveAll(s3.FileSystem.getFullPath(option.GetPath())); err != nil {
			return InternalError()
		}
		return nil
	}
	match := option.GetOption("if-match")
	if match != "" {
		etag, s3err := s3.getETag(option.GetPath())
		if s3err != nil {
			return nil
		}
		if match != etag {
			return PreconditionFailed()
		}
	}
	if !s3.FileSystem.deleteFile(option.GetPath()) {
		return InternalError()
	}
	return nil
}

func (s3 *S3Service) DeleteObjects(option S3Options) (*model.DeleteObjectsResult, *S3Error) {
	s := model.DeleteObjectsState{}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if !s3.FileSystem.isFileExists(option.GetBucket()) {
		return nil, NoSuchBucket()
	}
	if err := xml.Unmarshal(option.GetBody(), &s.ReqBody); err != nil {
		return nil, InternalError()
	}
	for _, obj := range s.ReqBody.Objects {
		s.DeleteList = append(s.DeleteList, obj.Key)
	}
	if len(s.DeleteList) == 0 || len(s.DeleteList) > 1000 {
		return nil, InternalError()
	}
	deleted, errors := s3.FileSystem.deleteFiles(option.GetBucket(), s.DeleteList)
	for _, v := range deleted {
		s.Deleted = append(s.Deleted, model.ObjectKey{Key: v})
	}
	for _, v := range errors {
		s.Error = append(s.Error, model.ObjectKey{Key: v})
	}
	result := s.MakeDeleteObjectsOptionsResult()
	return result, nil
}

func (s3 *S3Service) DeleteObjectTagging(option S3Options) *S3Error {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return InvalidBucketName()
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return KeyTooLongError()
	}
	var s3err *S3Error
	if s3err = s3.isBucketAndKeyExists(option.GetBucket(), option.GetPath()); s3err != nil {
		return s3err
	}
	if !s3.FileSystem.deleteMetaFile(filepath.Join(option.GetBucket(), option.GetKey())) {
		return InternalError()
	}
	return nil
}

func (s3 *S3Service) GetObject(option S3Options) (*model.GetObjectResult, *S3Error) {
	s := model.GetObjectState{}
	if err := s3.validateGetObjectOptions(option); err != nil {
		return nil, err
	}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if err := s3.isBucketAndKeyExists(option.GetBucket(), option.GetPath()); err != nil {
		return nil, err
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return nil, KeyTooLongError()
	}
	if s.Content = s3.FileSystem.readFile(option.GetPath()); s.Content == nil {
		return nil, InternalError()
	}
	var s3err *S3Error
	if s.Info, s.ETag, s3err = s3.getFileInfoAndETag(option.GetPath()); s3err != nil {
		return nil, s3err
	}
	if s3err = s3.validateETagAndTime(option); s3err != nil {
		return nil, s3err
	}
	if s.ContentRange, s.Content, s3err = s3.getRangeContent(option, s.Content); s3err != nil {
		return nil, s3err
	}
	if !s3.getPartNumberContent(option) {
		return nil, InvalidArgument()
	}
	if s.ResponseCrc64nvme, s3err = s3.checkChecksumMode(option, s.Content); s3err != nil {
		return nil, s3err
	}
	s.TagCount, s.MissingMeta = s3.Tag.getTagCount(option.GetPath())
	result := s.MakeGetObjectResult()
	result.ContentDisposition = option.GetOption("response-content-disposition")
	result.ContentEncoding = option.GetOption("response-content-encoding")
	result.ContentLanguage = option.GetOption("response-content-language")
	result.ContentType = option.GetOption("response-content-type")
	return result, nil
}

func (s3 *S3Service) GetObjectAttributes(option S3Options) (*model.GetObjectAttributesResult, *S3Error) {
	allowed := []string{"ETag", "Checksum", "ObjectParts", "StorageClass", "ObjectSize", ","}
	s := model.GetObjectAttributesState{Allowed: allowed}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if err := s3.isBucketAndKeyExists(option.GetBucket(), option.GetPath()); err != nil {
		return nil, err
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return nil, KeyTooLongError()
	}
	if err := s3.validateGetObjectAttributesOptions(option, s.Allowed); err != nil {
		return nil, err
	}
	var s3err *S3Error
	if s.Info, s3err = s3.getFileInfo(option.GetPath()); s3err != nil {
		return nil, s3err
	}
	if v := option.GetOption("x-amz-max-parts"); v != "" {
		s.MaxParts = v
	}
	if v := option.GetOption("x-amz-part-number-marker"); v != "" {
		if s.Marker = utils.ToInt(v); s.Marker == 0 {
			return nil, InternalError()
		}
	}
	v := option.GetOption("x-amz-object-attributes")
	if s.ETag, s3err = s3.etagIfNeeded(option.GetPath(), v); s3err != nil {
		return nil, InternalError()
	}
	if s.Checksum, s3err = s3.checksumIfNeeded(option.GetPath(), v); s3err != nil {
		return nil, InternalError()
	}
	if strings.Contains(v, "ObjectParts") {
		s.ObjectParts = true
	}
	if strings.Contains(v, "StorageClass") {
		s.StorageClass = "STANDARD"
	}
	if strings.Contains(v, "ObjectSize") {
		s.ObjectSize = s.Info.Size()
	}
	result := s.MakeGetObjectAttributesResult()
	return result, nil
}

func (s3 *S3Service) GetObjectTagging(option S3Options) (*model.Tagging, *S3Error) {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if err := s3.isBucketAndKeyExists(option.GetBucket(), option.GetPath()); err != nil {
		return nil, err
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return nil, KeyTooLongError()
	}
	tag := s3.Tag.readTag(option.GetPath())
	if tag == nil {
		return nil, InternalError()
	}
	return tag, nil
}

func (s3 *S3Service) HeadBucket(option S3Options) ([]byte, *S3Error) {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if b := option.GetBucket(); !s3.FileSystem.isFileExists(b) {
		return nil, NoSuchBucket()
	}
	return []byte{}, nil
}

func (s3 *S3Service) HeadObject(option S3Options) (*model.GetObjectResult, *S3Error) {
	s := model.GetObjectState{}
	if err := s3.validateGetObjectOptions(option); err != nil {
		return nil, err
	}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	path := option.GetPath()
	if err := s3.isBucketAndKeyExists(option.GetBucket(), path); err != nil {
		return nil, err
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return nil, KeyTooLongError()
	}
	if s.Content = s3.FileSystem.readFile(path); s.Content == nil {
		return nil, InternalError()
	}
	var s3err *S3Error
	if s.Info, s.ETag, s3err = s3.getFileInfoAndETag(path); s3err != nil {
		return nil, s3err
	}
	if s3err = s3.validateETagAndTime(option); s3err != nil {
		return nil, s3err
	}
	if s.ContentRange, s.Content, s3err = s3.getRangeContent(option, s.Content); s3err != nil {
		return nil, s3err
	}
	if !s3.getPartNumberContent(option) {
		return nil, InvalidArgument()
	}
	if s.ResponseCrc64nvme, s3err = s3.checkChecksumMode(option, s.Content); s3err != nil {
		return nil, s3err
	}
	result := s.MakeHeadObjectResult()
	result.ContentDisposition = option.GetOption("response-content-disposition")
	result.ContentEncoding = option.GetOption("response-content-encoding")
	result.ContentLanguage = option.GetOption("response-content-language")
	result.ContentType = option.GetOption("response-content-type")
	return result, nil
}

func (s3 *S3Service) ListBuckets(option S3Options) (*model.ListBucketsResult, *S3Error) {
	s := model.ListBucketsState{MaxBuckets: 10000}
	s.Dirs = utils.GetDirOnly(s3.FileSystem.readDir())
	if v := option.GetOption("continuation-token"); v != "" {
		if s.Dirs = s3.FileSystem.continuationBucket(v, s.Dirs); s.Dirs == nil {
			return nil, InvalidArgument()
		}
	}
	if v := option.GetOption("prefix"); v != "" {
		s.Prefix = v
		s.Dirs = s3.FileSystem.prefixBucket(v, s.Dirs)
	}
	if v := option.GetOption("max-buckets"); v != "" {
		s.Dirs, s.ContinuationToken = s3.FileSystem.checkBuckets(v, s.Dirs, s.MaxBuckets)
	}
	if v := utils.LimitCheck(s.Dirs); v != "" {
		s.ContinuationToken = v
	}
	result := s.MakeListBucketsResult()
	return result, nil
}

func (s3 *S3Service) ListMultipartUploads(option S3Options) (*model.ListMultipartUploadsResult, *S3Error) {
	s := model.ListMultipartUploadsState{MaxUploads: 1000}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	s.Bucket = option.GetBucket()
	if !s3.FileSystem.isFileExists(s.Bucket) {
		return nil, NoSuchBucket()
	}
	s.Dirs = utils.GetDirOnly(s3.MultiPart.getMpDirInfo())
	if v := option.GetOption("max-uploads"); v != "" {
		s.MaxUploads = utils.ToInt(v)
	}
	if v := option.GetOption("prefix"); v != "" {
		s.Prefix = v
	}
	if v := option.GetOption("delimiter"); v == "/" {
		s.Delimiter = filepath.FromSlash(v)
	}
	var s3err *S3Error
	if s.URLFlag, s3err = s3.checkEncodingType(option); s3err != nil {
		return nil, s3err
	}
	if v := option.GetOption("key-marker"); v != "" {
		if v2 := option.GetOption("upload-id-marker"); v2 != "" { // key-markerが指定されていない場合、upload-id-markerは無視
			if s.UploadIDMarker, s.Target = s3.FileSystem.decodeContinuationToken(v2); s.Target == 0 {
				return nil, InternalError()
			}
		}
		s.KeyMarker = v // keyの読み取り
	}
	res, responseRes := s3.listMpUploads(s)
	result := s.MakeListMultipartUploadsResult(*responseRes)
	result.Upload = *res
	return result, nil
}

func (s3 *S3Service) ListObjects(option S3Options) (*model.ListObjectsResult, *S3Error) {
	s := model.ListObjectsState{MaxKeys: 1000}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	s.Bucket = option.GetBucket()
	if !s3.FileSystem.isFileExists(s.Bucket) {
		return nil, NoSuchBucket()
	}
	s.BucketAPath = s3.FileSystem.getFullPath(s.Bucket)
	if v := option.GetOption("max-keys"); v != "" {
		if s.MaxKeys = utils.ToInt(v); s.MaxKeys > 1000 { // max-keysの上限は1000
			s.MaxKeys = 1000
		}
	}
	if v := option.GetOption("prefix"); v != "" {
		s.Prefix = v
	}
	if v := option.GetOption("marker"); v != "" {
		s.Marker = strings.ReplaceAll(v, "/", "\\")
	}
	if v := option.GetOption("delimiter"); v == "/" {
		s.Delimiter = filepath.FromSlash(v)
	}
	var s3err *S3Error
	if s.URLFlag, s3err = s3.checkEncodingType(option); s3err != nil {
		return nil, s3err
	}
	res, responseRes := s3.listObjects(s)
	if res == nil {
		return nil, NotImplemented()
	}
	result := s.MakeListObjectsResult(*responseRes)
	result.Contents = *res
	return result, nil
}

func (s3 *S3Service) ListObjectsV2(option S3Options) (*model.ListObjectsV2Result, *S3Error) {
	s := model.ListObjectsState{MaxKeys: 1000, V2Flg: true}
	s.Bucket = option.GetBucket()
	if !s3.FileSystem.checkBucketName(s.Bucket) {
		return nil, InvalidBucketName()
	}
	if !s3.FileSystem.isFileExists(s.Bucket) {
		return nil, NoSuchBucket()
	}
	s.BucketAPath = s3.FileSystem.getFullPath(s.Bucket)
	if v := option.GetOption("max-keys"); v != "" {
		if s.MaxKeys = utils.ToInt(v); s.MaxKeys > 1000 { // max-keysの上限は1000
			s.MaxKeys = 1000
		}
	}
	if v := option.GetOption("prefix"); v != "" {
		s.Prefix = v
	}
	if v := option.GetOption("start-after"); v != "" {
		v = strings.ReplaceAll(v, "/", "\\")
		s.StartAfter = v
	}
	if v := option.GetOption("delimiter"); v == "/" {
		s.Delimiter = filepath.FromSlash(v)
	}
	var s3err *S3Error
	if s.URLFlag, s3err = s3.checkEncodingType(option); s3err != nil {
		return nil, s3err
	}
	if v := option.GetOption("continuation-token"); v != "" {
		if s.ContinuationToken, s.Target = s3.FileSystem.decodeContinuationToken(v); s.Target == 0 {
			return nil, InternalError()
		}
	}
	res, responseRes := s3.listObjects(s)
	if res == nil {
		return nil, NotImplemented()
	}
	result := s.MakeListObjectsV2Result(*responseRes)
	result.Contents = *res
	return result, nil
}

func (s3 *S3Service) ListParts(option S3Options) (*model.ListPartsResult, *S3Error) {
	s := model.ListPartsState{MaxParts: 1000}
	s.Bucket = option.GetBucket()
	if !s3.FileSystem.checkBucketName(s.Bucket) {
		return nil, InvalidBucketName()
	}
	if !s3.FileSystem.isFileExists(s.Bucket) {
		return nil, NoSuchBucket()
	}
	s.UploadID = option.GetOption("uploadId")
	if !s3.FileSystem.isFileExists(s3.FileSystem.getUploadIDPath(s.UploadID)) {
		return nil, NoSuchUpload()
	}
	s.BucketAPath = s3.FileSystem.getFullPath(s3.FileSystem.getUploadIDPath(s.UploadID))
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return nil, KeyTooLongError()
	}
	if s.Key = s3.MultiPart.getMpUploadMeta(option.GetKey(), s.UploadID); s.Key == "" {
		return nil, NoSuchUpload()
	}
	if v := option.GetOption("max-parts"); v != "" {
		s.MaxParts = utils.ToInt(v)
	}
	if v := option.GetOption("part-number-marker"); v != "" {
		if s.Target = utils.ToInt(v); s.Target == 0 {
			return nil, InternalError()
		}
	}
	res, responseRes := s3.listParts(s)
	if res == nil {
		return nil, NotImplemented()
	}
	result := s.MakeListPartsResult(*responseRes)
	result.Part = *res
	return result, nil
}

func (s3 *S3Service) PutObject(option S3Options) (*model.PutObjectResult, *S3Error) {
	s := model.PutObjectState{}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if b := option.GetBucket(); !s3.FileSystem.isFileExists(b) {
		return nil, NoSuchBucket()
	}
	if err := s3.validateOptions(option); err != nil {
		return nil, err
	}
	if f := s3.FileSystem.uploadFile(option.GetBody(), option.GetPath()); !f {
		return nil, InternalError()
	}
	if err := s3.Tag.putTagging(option, option.GetPath()); err != nil {
		return nil, err
	}
	var s3err *S3Error
	if s.ETag, s3err = s3.getETag(option.GetPath()); s3err != nil {
		return nil, InternalError()
	}
	if s3err = s3.compareMd5(option, nil, s.ETag); s3err != nil {
		return nil, s3err
	}
	if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.getChecksumMode(option, ""); s3err != nil {
		return nil, s3err
	}
	result := s.MakePutObjectResult()
	return result, nil
}

func (s3 *S3Service) PutObjectTagging(option S3Options) *S3Error {
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return InvalidBucketName()
	}
	var s3err *S3Error
	if s3err = s3.isBucketAndKeyExists(option.GetBucket(), option.GetPath()); s3err != nil {
		return s3err
	}
	body := option.GetBody()
	var reqBody model.Tagging
	if err := xml.Unmarshal(body, &reqBody); err != nil {
		return InvalidRequest()
	}
	if !s3.FileSystem.validateChecksumAlgorithm(option.GetOption("x-amz-sdk-checksum-algorithm")) { // 値チェックのみ
		return InvalidArgument()
	}
	if s3err = s3.compareMd5(option, body, ""); s3err != nil {
		return s3err
	}
	if s3err = s3.Tag.putOnlyTagging(reqBody, option.GetPath()); s3err != nil {
		return s3err
	}
	return nil
}

func (s3 *S3Service) UploadPart(option S3Options) (*model.PutObjectResult, *S3Error) {
	s := model.PutObjectState{}
	partNum := option.GetOption("partNumber")
	if !utils.CheckPartNumber(partNum) {
		return nil, InvalidArgument()
	}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	if b := option.GetBucket(); !s3.FileSystem.isFileExists(b) {
		return nil, NoSuchBucket()
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return nil, KeyTooLongError()
	}
	if !s3.FileSystem.canCreateFile(s3.FileSystem.getFullPath(option.GetPath())) {
		return nil, InternalError()
	}
	id := option.GetOption("uploadId")
	if !s3.FileSystem.isFileExists(s3.FileSystem.getUploadIDPath(id)) {
		return nil, NoSuchUpload()
	}
	pNumPath := s3.FileSystem.getPartNumberPath(id, partNum)
	if f := s3.FileSystem.uploadFile(option.GetBody(), pNumPath); !f {
		return nil, InternalError()
	}
	var s3err *S3Error
	if s.ETag, s3err = s3.getETag(pNumPath); s3err != nil {
		return nil, InternalError()
	}
	if s3err = s3.compareMd5(option, nil, s.ETag); s3err != nil {
		return nil, s3err
	}
	if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.getChecksumMode(option, pNumPath); s3err != nil {
		return nil, s3err
	}
	result := s.MakePutObjectResult()
	return result, nil
}

func (s3 *S3Service) UploadPartCopy(option S3Options) (*model.CopyObjectResult, *S3Error) {
	s := model.CopyObjectState{}
	partNum := option.GetOption("partNumber")
	if !utils.CheckPartNumber(partNum) {
		return nil, InvalidArgument()
	}
	s.SrcPath = option.GetOption("x-amz-copy-source")
	if s.SrcPath == "" {
		return nil, InvalidArgument()
	}
	if !s3.FileSystem.checkBucketName(option.GetBucket()) {
		return nil, InvalidBucketName()
	}
	var s3err *S3Error
	if s3err = s3.isBucketAndKeyExists(option.GetBucket(), s.SrcPath); s3err != nil {
		return nil, s3err
	}
	if !s3.FileSystem.checkKeyName(option.GetKey()) {
		return nil, KeyTooLongError()
	}
	if !s3.FileSystem.canCreateFile(s3.FileSystem.getFullPath(option.GetPath())) {
		return nil, InternalError()
	}
	id := option.GetOption("uploadId")
	if key := s3.MultiPart.getMpUploadMeta(option.GetKey(), id); key == "" {
		return nil, NoSuchKey()
	}
	if !s3.needsCopy(option, s.SrcPath) {
		return nil, PreconditionFailed()
	}
	if s.Offset, s.Length = s3.getRangeValue(option); s.Offset == -1 {
		return nil, RangeNotSatisfiable()
	}
	s.DstPath = s3.FileSystem.getPartNumberPath(id, partNum)
	if !s3.FileSystem.copyFile(s.SrcPath, s.DstPath, s.Offset, s.Length) {
		return nil, InternalError()
	}
	if s.Info, s.ETag, s3err = s3.getFileInfoAndETag(s.DstPath); s3err != nil {
		return nil, s3err
	}
	a := option.GetOption("x-amz-checksum-algorithm")
	if !s3.FileSystem.validateChecksumAlgorithm(a) {
		return nil, InvalidArgument()
	}
	if s.ChecksumAlgorithm, s.ChecksumValue, s3err = s3.compareChecksum(option, s.SrcPath, a, "", true); s3err != nil {
		return nil, s3err
	}
	result := s.MakeCopyObjectResult()
	return result, nil
}
