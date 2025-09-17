// Copyright 2025-2025 RIKEN R-CCS\.
// SPDX-License-Identifier: BSD-2-Clause

package api

import (
	"net/http"
	"s3-baby-server/internal/service"
)

func PutObjectHandler(s3 *service.S3Service) S3HandlerFunc {
	return func(w http.ResponseWriter, options *HTTPS3Options) *service.S3Error {
		if options.GetKey() == "" { // keyが空ならCreateBucketの処理を呼び出し
			CreateBucketHandler(s3)
			return nil
		}
		options.Logger.Info("PutObject")
		result, err := s3.PutObject(options)
		if err != nil {
			return err
		}
		ResponsePutHeader(result, w, *options, http.StatusOK)
		return nil
	}
}
