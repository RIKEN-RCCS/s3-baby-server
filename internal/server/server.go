// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"net/http"
	"s3-baby-server/internal/api"
	"s3-baby-server/internal/service"

	"github.com/gorilla/mux"
)

// MultiRoute は下記の複数ルートを管理するためのヘルパ構造体。
// gorillaがAPIパス末尾の「/」のありなし区別をしてしまうため、
// 「/」ありとなしの2ルートを同じハンドラにバインドする。
type MultiRoute struct {
	routes []*mux.Route
}

func (multi *MultiRoute) Methods(methods ...string) *MultiRoute {
	for _, r := range multi.routes {
		r.Methods(methods...)
	}
	return multi
}

func (multi *MultiRoute) HeadersRegexp(pairs ...string) *MultiRoute {
	for _, r := range multi.routes {
		r.HeadersRegexp(pairs...)
	}
	return multi
}

func (multi *MultiRoute) Queries(pairs ...string) *MultiRoute {
	for _, r := range multi.routes {
		r.Queries(pairs...)
	}
	return multi
}

func Start(basePath, addr, logPath, authKey string) {
	r := mux.NewRouter()
	r.Use(PanicRecovery)
	logger := Init(logPath) // ログファイルの作成
	logger.Info("Starting server", "address", addr)
	logger.Debug("options", "authKey", authKey)
	fs := &service.FileSystem{Logger: logger, RootPath: basePath, TmpPath: "/.S3BabyServer/TmpUpload", MpPath: "/.S3BabyServer/MultipartUpload"}
	mp := &service.MultiPart{FileSystem: fs}
	t := &service.Tag{FileSystem: fs, DirectiveCopy: "COPY", DirectiveReplace: "REPLACE"}
	s3 := &service.S3Service{FileSystem: fs, MultiPart: mp, Tag: t}
	fs.InitDir()
	bind := func(path string, f api.S3HandlerFunc) *mux.Route {
		return r.HandleFunc(path, api.HandlerBase(f, s3.FileSystem, authKey, logger))
	}
	multiBind := func(path string, f api.S3HandlerFunc) *MultiRoute {
		return &MultiRoute{routes: []*mux.Route{bind(path, f), bind(path+"/", f)}}
	}
	// 各APIのハンドラを登録
	multiBind("/{bucket}/{key:.*}", api.UploadPartCopyHandler(s3)).Methods("PUT").HeadersRegexp("x-amz-copy-source", ".*").Queries("partNumber", "").Queries("uploadId", "")
	multiBind("/{bucket}/{key:.*}", api.CopyObjectHandler(s3)).Methods("PUT").HeadersRegexp("x-amz-copy-source", ".*")
	multiBind("/{bucket}", api.DeleteObjectsHandler(s3)).Methods("POST").Queries("delete", "")
	multiBind("/{bucket}", api.ListObjectsV2Handler(s3)).Methods("GET").Queries("list-type", "2")
	multiBind("/{bucket}/{key:.*}", api.AbortMultipartUploadHandler(s3)).Methods("DELETE").Queries("uploadId", "")
	multiBind("/{bucket}/{key:.*}", api.CompleteMultipartUploadHandler(s3)).Methods("POST").Queries("uploadId", "")
	multiBind("/{bucket}/{key:.*}", api.CreateMultipartUploadHandler(s3)).Methods("POST").Queries("uploads", "")
	multiBind("/{bucket}/{key:.*}", api.DeleteObjectTaggingHandler(s3)).Methods("DELETE").Queries("tagging", "")
	multiBind("/{bucket}/{key:.*}", api.GetObjectAttributesHandler(s3)).Methods("GET").Queries("attributes", "")
	multiBind("/{bucket}/{key:.*}", api.GetObjectTaggingHandler(s3)).Methods("GET").Queries("tagging", "")
	multiBind("/{bucket}", api.ListMultipartUploadsHandler(s3)).Methods("GET").Queries("uploads", "")
	multiBind("/{bucket}/{key:.*}", api.ListPartsHandler(s3)).Methods("GET").Queries("uploadId", "")
	multiBind("/{bucket}/{key:.*}", api.PutObjectTaggingHandler(s3)).Methods("PUT").Queries("tagging", "")
	multiBind("/{bucket}/{key:.*}", api.UploadPartHandler(s3)).Methods("PUT").Queries("partNumber", "").Queries("uploadId", "")
	multiBind("/{bucket}", api.DeleteBucketHandler(s3)).Methods("DELETE")
	multiBind("/{bucket}/{key:.*}", api.DeleteObjectHandler(s3)).Methods("DELETE")
	multiBind("/{bucket}", api.CreateBucketHandler(s3)).Methods("PUT")
	multiBind("/{bucket}/{key:.*}", api.PutObjectHandler(s3)).Methods("PUT")
	multiBind("/{bucket}", api.ListObjectsHandler(s3)).Methods("GET")
	multiBind("/{bucket}/{key:.*}", api.GetObjectHandler(s3)).Methods("GET")
	multiBind("/{bucket}", api.HeadBucketHandler(s3)).Methods("HEAD")
	multiBind("/{bucket}/{key:.*}", api.HeadObjectHandler(s3)).Methods("HEAD")
	multiBind("/", api.ListBucketsHandler(s3)).Methods("GET")
	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Error("", "error", err)
	}
}
