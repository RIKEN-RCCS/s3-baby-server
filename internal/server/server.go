// Copyright 2025-2025 RIKEN R-CCS.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"s3-baby-server/internal/api"
	"s3-baby-server/internal/service"
	"time"
)

type prior_handler struct {
	bbs *Bb_server
	sx *http.ServeMux
}

// PRIOR_HANDLER checks an authorization header in a request before
// passing it to actual handlers.
func (sv *prior_handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//var logger = bbs.Logger
	//var authKey = bbs.AuthKey

	//option := newHTTPS3Options(r, logger)
	//if !option.checkAuthorization(r, authKey) {
	//if !option.CheckErrorHeader() {
	//if !option.CheckKeyPath(s3.RootPath, option.GetPath()) {

	fmt.Printf("prior_handler does nothing.\n")
	sv.sx.ServeHTTP(w, r)

	//option.Logger.Error(err.Message, "status code", err.Status)
	//w.Header().Set("Content-Type", "application/xml")
	//w.WriteHeader(err.Status)
	//if err := xml.NewEncoder(w).Encode(err); err != nil {
}

// SERVER_CONTROL handles requests to shutdown.  It is hooked at the
// url "/bbs.ctl".
func (bbs *Bb_server) server_control(w http.ResponseWriter, r *http.Request) {
	var q = r.URL.Query()
	var delete = q.Has("delete")
	if delete {
		log.Fatal("SHUTDOWN")
	}
}

func Start(basePath, addr, logPath, authKey string) {
	logger := Init(logPath)

	// Run in UTC time zone instead of local time zone.

	time.Local = time.UTC

	// Convert a path to platform specific one.

	//var basepath1 = file.Clean(basePath)
	var basepath2 = filepath.Clean(basePath)

	//r := mux.NewRouter()
	//r.Use(PanicRecovery)
	var sx = http.NewServeMux()

	//logger := Init(logPath)
	logger.Info("Starting server", "address", addr)
	logger.Debug("options", "authKey", authKey)
	fs := &service.FileSystem{Logger: logger, RootPath: basepath2, TmpPath: "/.S3BabyServer/TmpUpload", MpPath: "/.S3BabyServer/MultipartUpload"}
	mp := &service.MultiPart{FileSystem: fs}
	t := &service.Tag{FileSystem: fs, DirectiveCopy: "COPY", DirectiveReplace: "REPLACE"}
	s3 := &service.S3Service{FileSystem: fs, MultiPart: mp, Tag: t}
	fs.InitDir()

	var bbs = Bb_server{S3: s3, Logger: logger, AuthKey: authKey}

	//bind := func(path string, f api.S3HandlerFunc) *mux.Route {
	//	return r.HandleFunc(path, api.HandlerBase(f, s3.FileSystem, authKey, logger))
	//}
	//multiBind := func(path string, f api.S3HandlerFunc) *MultiRoute {
	//	return &MultiRoute{routes: []*mux.Route{bind(path, f), bind(path+"/", f)}}
	//}
	// 各APIのハンドラを登録
	//multiBind("/{bucket}/{key:.*}", api.UploadPartCopyHandler(s3)).Methods("PUT").HeadersRegexp("x-amz-copy-source", ".*").Queries("partNumber", "").Queries("uploadId", "")
	//multiBind("/{bucket}/{key:.*}", api.CopyObjectHandler(s3)).Methods("PUT").HeadersRegexp("x-amz-copy-source", ".*")
	//multiBind("/{bucket}", api.DeleteObjectsHandler(s3)).Methods("POST").Queries("delete", "")
	//multiBind("/{bucket}", api.ListObjectsV2Handler(s3)).Methods("GET").Queries("list-type", "2")
	//multiBind("/{bucket}/{key:.*}", api.AbortMultipartUploadHandler(s3)).Methods("DELETE").Queries("uploadId", "")
	//multiBind("/{bucket}/{key:.*}", api.CompleteMultipartUploadHandler(s3)).Methods("POST").Queries("uploadId", "")
	//multiBind("/{bucket}/{key:.*}", api.CreateMultipartUploadHandler(s3)).Methods("POST").Queries("uploads", "")
	//multiBind("/{bucket}/{key:.*}", api.DeleteObjectTaggingHandler(s3)).Methods("DELETE").Queries("tagging", "")
	//multiBind("/{bucket}/{key:.*}", api.GetObjectAttributesHandler(s3)).Methods("GET").Queries("attributes", "")
	//multiBind("/{bucket}/{key:.*}", api.GetObjectTaggingHandler(s3)).Methods("GET").Queries("tagging", "")
	//multiBind("/{bucket}", api.ListMultipartUploadsHandler(s3)).Methods("GET").Queries("uploads", "")
	//multiBind("/{bucket}/{key:.*}", api.ListPartsHandler(s3)).Methods("GET").Queries("uploadId", "")
	//multiBind("/{bucket}/{key:.*}", api.PutObjectTaggingHandler(s3)).Methods("PUT").Queries("tagging", "")
	//multiBind("/{bucket}/{key:.*}", api.UploadPartHandler(s3)).Methods("PUT").Queries("partNumber", "").Queries("uploadId", "")
	//multiBind("/{bucket}", api.DeleteBucketHandler(s3)).Methods("DELETE")
	//multiBind("/{bucket}/{key:.*}", api.DeleteObjectHandler(s3)).Methods("DELETE")
	//multiBind("/{bucket}", api.CreateBucketHandler(s3)).Methods("PUT")
	//multiBind("/{bucket}/{key:.*}", api.PutObjectHandler(s3)).Methods("PUT")
	//multiBind("/{bucket}", api.ListObjectsHandler(s3)).Methods("GET")
	//multiBind("/{bucket}/{key:.*}", api.GetObjectHandler(s3)).Methods("GET")
	//multiBind("/{bucket}", api.HeadBucketHandler(s3)).Methods("HEAD")
	//multiBind("/{bucket}/{key:.*}", api.HeadObjectHandler(s3)).Methods("HEAD")
	//multiBind("/", api.ListBucketsHandler(s3)).Methods("GET")

	bbs.AbortMultipartUploadHandler = api.HandlerBase(api.AbortMultipartUploadHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.CompleteMultipartUploadHandler = api.HandlerBase(api.CompleteMultipartUploadHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.CopyObjectHandler = api.HandlerBase(api.CopyObjectHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.CreateBucketHandler = api.HandlerBase(api.CreateBucketHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.CreateMultipartUploadHandler = api.HandlerBase(api.CreateMultipartUploadHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.DeleteBucketHandler = api.HandlerBase(api.DeleteBucketHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.DeleteObjectHandler = api.HandlerBase(api.DeleteObjectHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.DeleteObjectsHandler = api.HandlerBase(api.DeleteObjectsHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.DeleteObjectTaggingHandler = api.HandlerBase(api.DeleteObjectTaggingHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.GetObjectAttributesHandler = api.HandlerBase(api.GetObjectAttributesHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.GetObjectHandler = api.HandlerBase(api.GetObjectHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.GetObjectTaggingHandler = api.HandlerBase(api.GetObjectTaggingHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.HeadBucketHandler = api.HandlerBase(api.HeadBucketHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.HeadObjectHandler = api.HandlerBase(api.HeadObjectHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.ListBucketsHandler = api.HandlerBase(api.ListBucketsHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.ListMultipartUploadsHandler = api.HandlerBase(api.ListMultipartUploadsHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.ListObjectsHandler = api.HandlerBase(api.ListObjectsHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.ListObjectsV2Handler = api.HandlerBase(api.ListObjectsV2Handler(s3),
		s3.FileSystem, authKey, logger)
	bbs.ListPartsHandler = api.HandlerBase(api.ListPartsHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.PutObjectHandler = api.HandlerBase(api.PutObjectHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.PutObjectTaggingHandler = api.HandlerBase(api.PutObjectTaggingHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.UploadPartCopyHandler = api.HandlerBase(api.UploadPartCopyHandler(s3),
		s3.FileSystem, authKey, logger)
	bbs.UploadPartHandler = api.HandlerBase(api.UploadPartHandler(s3),
		s3.FileSystem, authKey, logger)

	sx.HandleFunc("POST /bbs.ctl/{$}", func(w http.ResponseWriter, r *http.Request) {
		bbs.server_control(w, r)
	})

	register_dispatcher(&bbs, sx)
	var sv = prior_handler{&bbs, sx}
	if err := http.ListenAndServe(addr, &sv); err != nil {
		logger.Error("", "error", err)
	}
}
