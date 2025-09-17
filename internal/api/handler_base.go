// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"encoding/xml"
	"log/slog"
	"net/http"
	"s3-baby-server/internal/model"
	"s3-baby-server/internal/service"
	"strconv"
	"time"
)

type S3HandlerFunc func(http.ResponseWriter, *HTTPS3Options) *service.S3Error

func HandlerBase(handler S3HandlerFunc, s3 *service.FileSystem, authKey string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		option := newHTTPS3Options(r, logger)
		if !option.checkAuthorization(r, authKey) {
			http.Error(w, "The Access Key Id you provided does not exist in our records", http.StatusUnauthorized)
			return
		}
		if !option.CheckErrorHeader() {
			http.Error(w, "Invalid headers are specified", http.StatusBadRequest)
			return
		}
		if !option.CheckKeyPath(s3.RootPath, option.GetPath()) {
			http.Error(w, "Check the name of the key", http.StatusBadRequest)
			return
		}
		if err := handler(w, option); err != nil {
			option.Logger.Error(err.Message, "status code", err.Status)
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(err.Status)
			if err := xml.NewEncoder(w).Encode(err); err != nil {
				option.Logger.Error("", "error", err)
			}
		}
	}
}

func ResponseWriteHeader(result any, w http.ResponseWriter, option HTTPS3Options, status int) {
	w.Header().Set("Content-Type", "application/xml")
	res, err := xml.MarshalIndent(result, " ", "  ")
	if err != nil {
		option.Logger.Error("", "error", err)
	}
	ResponseStatusHeader(w, option, status)
	if _, err = w.Write(res); err != nil {
		option.Logger.Error("", "error", err)
	}
}

func ResponseStatusHeader(w http.ResponseWriter, option HTTPS3Options, status int) {
	w.WriteHeader(status)
	option.Logger.Info("", "status code", status)
}

func ResponsePutHeader(result any, w http.ResponseWriter, option HTTPS3Options, status int) {
	output, _ := xml.MarshalIndent(result, " ", "  ")
	v := model.PutObjectResult{}
	if err := xml.Unmarshal(output, &v); err != nil {
		option.Logger.Error("", "error", err)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Etag", v.ETag)
	headers := map[string]string{
		"x-amz-checksum-crc32":     v.ChecksumCRC32,
		"x-amz-checksum-crc32c":    v.ChecksumCRC32C,
		"x-amz-checksum-crc64nvme": v.ChecksumCRC64NVME,
		"x-amz-checksum-sha1":      v.ChecksumSHA1,
		"x-amz-checksum-sha256":    v.ChecksumSHA256,
	}
	for key, value := range headers {
		if value != "" {
			w.Header().Set(key, value)
			w.Header().Set("X-Amz-Checksum-Type", v.ChecksumType)
		}
	}
	ResponseStatusHeader(w, option, status)
}

func ResponseGetHeader(result *model.GetObjectResult, w http.ResponseWriter, option HTTPS3Options, status int, writeFlg bool) {
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Last-Modified", result.LastModified.Format(time.RFC3339))
	w.Header().Set("Content-Length", strconv.Itoa(result.ContentLength))
	w.Header().Set("Etag", result.ETag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Amz-Storage-Class", "STANDARD")
	headers := map[string]string{
		"Content-Range":            result.ContentRange,
		"Content-Disposition":      result.ContentDisposition,
		"Content-Encoding":         result.ContentEncoding,
		"Content-Language":         result.ContentLanguage,
		"Content-Type":             result.ContentType,
		"x-amz-checksum-crc64nvme": result.CRC64NVME,
		"x-amz-checksum-type":      result.ChecksumType,
		"x-amz-tagging-count":      result.TagCount,
	}
	for key, value := range headers {
		if value != "" {
			w.Header().Set(key, value)
			if key == "Content-Range" {
				status = http.StatusPartialContent // Range, partNumber成功；206 Partial Content
			}
		}
	}
	ResponseStatusHeader(w, option, status)
	if writeFlg {
		if _, err := w.Write(result.Content); err != nil {
			option.Logger.Error("failed to write response", "error", err)
		}
	}
}

func ResponseCreateMultiHeader(result *model.CreateMultipartUploadResult, w http.ResponseWriter, option HTTPS3Options, status int) {
	output, _ := xml.MarshalIndent(result.InitiateMultipartUploadResult, " ", "  ")
	v := model.CreateMultipartUploadResult{}
	if err := xml.Unmarshal(output, &v.InitiateMultipartUploadResult); err != nil {
		option.Logger.Error("", "error", err)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "application/xml")
	if result.ChecksumAlgorithm != "" {
		w.Header().Set("X-Amz-Checksum-Algorithm", result.ChecksumAlgorithm)
		w.Header().Set("X-Amz-Checksum-Type", result.ChecksumType)
	}
	ResponseStatusHeader(w, option, status)
	if _, err := w.Write(output); err != nil {
		option.Logger.Error("", "error", err)
	}
}
